package localmetrics

import (
	"database/sql"
	"net"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/sqlite"
)

type GeoLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Region      string  `json:"regionName"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
}

type RouteMetricsSnapshot struct {
	RequestCount  int64
	ErrorCount    int64
	DurationAvgMs int64
	DurationMinMs int64
	DurationMaxMs int64
}

type MapPoint struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Success     int64   `json:"success"`
	Errors      int64   `json:"errors"`
	Blocked     int64   `json:"blocked"`
}

type TimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Success   int64     `json:"success"`
	Errors    int64     `json:"errors"`
	Blocked   int64     `json:"blocked"`
}

type PathStat struct {
	Path     string `json:"path"`
	Method   string `json:"method"`
	Hostname string `json:"hostname"`
	Count    int64  `json:"count"`
	Errors   int64  `json:"errors"`
}

type IPStat struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Success     int64   `json:"success"`
	Errors      int64   `json:"errors"`
	Blocked     int64   `json:"blocked"`
}

type Store struct {
	db sqlite.Database
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sqlite.OpenDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS access_log (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			bucket_start    INTEGER NOT NULL,
			hostname        TEXT    NOT NULL,
			client_ip       TEXT    NOT NULL,
			method          TEXT    NOT NULL,
			path            TEXT    NOT NULL DEFAULT '',
			request_count   INTEGER NOT NULL,
			error_count     INTEGER NOT NULL,
			duration_avg_ms INTEGER NOT NULL,
			duration_min_ms INTEGER NOT NULL,
			duration_max_ms INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_bucket   ON access_log(bucket_start)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_hostname ON access_log(hostname)`,
		`CREATE TABLE IF NOT EXISTS geo_cache (
			ip           TEXT PRIMARY KEY,
			lat          REAL,
			lon          REAL,
			country      TEXT,
			country_code TEXT,
			city         TEXT,
			region       TEXT,
			isp          TEXT,
			org          TEXT,
			updated_at   INTEGER NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	// Add path column to existing databases that predate this migration
	var hasPath int
	row := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('access_log') WHERE name='path'`)
	if row.Scan(&hasPath) == nil && hasPath == 0 {
		if _, err := s.db.Exec(`ALTER TABLE access_log ADD COLUMN path TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	// Normalize all local network entries to consistent values so they group as a single map point.
	// Catches variations: city='Localhost', country='Local', etc. from older code.
	if _, err := s.db.Exec(`UPDATE geo_cache SET lat = 30, lon = -40, country = 'Local Network', country_code = 'LC', city = 'Local'
		WHERE country IN ('Local Network', 'Local') OR city IN ('Local', 'Localhost')`); err != nil {
		return err
	}
	return nil
}

func (s *Store) RecordBatch(bucketStart time.Time, hostname, clientIP, method, path string, rm RouteMetricsSnapshot) error {
	_, err := s.db.Exec(
		`INSERT INTO access_log (bucket_start, hostname, client_ip, method, path, request_count, error_count, duration_avg_ms, duration_min_ms, duration_max_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bucketStart.Unix(), hostname, clientIP, method, path,
		rm.RequestCount, rm.ErrorCount, rm.DurationAvgMs, rm.DurationMinMs, rm.DurationMaxMs,
	)
	return err
}

func (s *Store) SetGeoLocation(ip string, geo GeoLocation) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO geo_cache (ip, lat, lon, country, country_code, city, region, isp, org, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ip, geo.Lat, geo.Lon, geo.Country, geo.CountryCode, geo.City, geo.Region, geo.ISP, geo.Org, time.Now().Unix(),
	)
	return err
}

func (s *Store) GetGeoLocation(ip string) (*GeoLocation, bool, error) {
	row := s.db.QueryRow(
		`SELECT lat, lon, country, country_code, city, region, isp, org FROM geo_cache WHERE ip = ?`, ip,
	)
	var geo GeoLocation
	err := row.Scan(&geo.Lat, &geo.Lon, &geo.Country, &geo.CountryCode, &geo.City, &geo.Region, &geo.ISP, &geo.Org)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &geo, true, nil
}

func (s *Store) GetMapData(from, to time.Time, hostname, clientIP string) ([]MapPoint, error) {
	// A hostname is "valid" (known/configured) if it ever had successful requests across all time.
	// Requests to unknown/blocked hostnames (including no-SNI stored as 'NONE') have request_count = error_count
	// in every row, so they never appear in the valid set and are classified as blocked.
	query := `WITH valid AS (
		SELECT DISTINCT hostname FROM access_log WHERE request_count > error_count
	)
	SELECT g.lat, g.lon, g.country, g.country_code, g.city,
		SUM(CASE WHEN v.hostname IS NULL THEN a.request_count ELSE 0 END) AS blocked,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN a.error_count ELSE 0 END) AS errors,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN MAX(0, a.request_count - a.error_count) ELSE 0 END) AS success
	FROM access_log a
	JOIN geo_cache g ON a.client_ip = g.ip
	LEFT JOIN valid v ON a.hostname = v.hostname
	WHERE a.bucket_start >= ? AND a.bucket_start <= ?`
	args := []any{from.Unix(), to.Unix()}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	if clientIP != "" {
		query += " AND a.client_ip = ?"
		args = append(args, clientIP)
	}
	query += " GROUP BY g.lat, g.lon, g.country, g.country_code, g.city ORDER BY SUM(a.request_count) DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []MapPoint
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(&p.Lat, &p.Lon, &p.Country, &p.CountryCode, &p.City, &p.Blocked, &p.Errors, &p.Success); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *Store) GetTimeSeries(from, to time.Time, hostname, granularity, clientIP, city, country string) ([]TimePoint, error) {
	var bucketSeconds int64 = 3600
	if granularity == "day" {
		bucketSeconds = 86400
	}
	hasLocation := city != "" && country != ""

	query := `WITH valid AS (
		SELECT DISTINCT hostname FROM access_log WHERE request_count > error_count
	)
	SELECT (a.bucket_start / ?) * ? as ts,
		SUM(CASE WHEN v.hostname IS NULL THEN a.request_count ELSE 0 END) AS blocked,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN a.error_count ELSE 0 END) AS errors,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN MAX(0, a.request_count - a.error_count) ELSE 0 END) AS success
	FROM access_log a
	LEFT JOIN valid v ON a.hostname = v.hostname`
	if hasLocation {
		query += ` JOIN geo_cache g ON a.client_ip = g.ip`
	}
	query += ` WHERE a.bucket_start >= ? AND a.bucket_start <= ?`
	args := []any{bucketSeconds, bucketSeconds, from.Unix(), to.Unix()}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	if clientIP != "" {
		query += " AND a.client_ip = ?"
		args = append(args, clientIP)
	}
	if hasLocation {
		query += " AND g.city = ? AND g.country = ?"
		args = append(args, city, country)
	}
	query += " GROUP BY ts ORDER BY ts"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TimePoint
	for rows.Next() {
		var ts int64
		var p TimePoint
		if err := rows.Scan(&ts, &p.Blocked, &p.Errors, &p.Success); err != nil {
			return nil, err
		}
		p.Timestamp = time.Unix(ts, 0).UTC()
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *Store) GetHostnames() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT hostname FROM access_log ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hostnames []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		hostnames = append(hostnames, h)
	}
	return hostnames, rows.Err()
}

func (s *Store) GetTopPaths(from, to time.Time, hostname, clientIP, city, country string, limit int) ([]PathStat, error) {
	hasLocation := city != "" && country != ""

	query := `SELECT a.path, a.method, a.hostname, SUM(a.request_count), SUM(a.error_count)
		FROM access_log a`
	if hasLocation {
		query += ` JOIN geo_cache g ON a.client_ip = g.ip`
	}
	query += ` WHERE a.bucket_start >= ? AND a.bucket_start <= ? AND a.path != ''`
	args := []any{from.Unix(), to.Unix()}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	if clientIP != "" {
		query += " AND a.client_ip = ?"
		args = append(args, clientIP)
	}
	if hasLocation {
		query += " AND g.city = ? AND g.country = ?"
		args = append(args, city, country)
	}
	query += " GROUP BY a.path, a.method, a.hostname ORDER BY SUM(a.request_count) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PathStat
	for rows.Next() {
		var p PathStat
		if err := rows.Scan(&p.Path, &p.Method, &p.Hostname, &p.Count, &p.Errors); err != nil {
			return nil, err
		}
		stats = append(stats, p)
	}
	return stats, rows.Err()
}

func (s *Store) GetIPStats(from, to time.Time, hostname string, limit int) ([]IPStat, error) {
	query := `WITH valid AS (
		SELECT DISTINCT hostname FROM access_log WHERE request_count > error_count
	)
	SELECT a.client_ip,
		COALESCE(g.country,''), COALESCE(g.country_code,''), COALESCE(g.city,''),
		COALESCE(g.lat,0), COALESCE(g.lon,0),
		SUM(CASE WHEN v.hostname IS NULL THEN a.request_count ELSE 0 END) AS blocked,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN a.error_count ELSE 0 END) AS errors,
		SUM(CASE WHEN v.hostname IS NOT NULL THEN MAX(0, a.request_count - a.error_count) ELSE 0 END) AS success
	FROM access_log a
	LEFT JOIN geo_cache g ON a.client_ip = g.ip
	LEFT JOIN valid v ON a.hostname = v.hostname
	WHERE a.bucket_start >= ? AND a.bucket_start <= ?`
	args := []any{from.Unix(), to.Unix()}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	query += " GROUP BY a.client_ip ORDER BY SUM(a.request_count) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []IPStat
	for rows.Next() {
		var p IPStat
		if err := rows.Scan(&p.IP, &p.Country, &p.CountryCode, &p.City, &p.Lat, &p.Lon, &p.Blocked, &p.Errors, &p.Success); err != nil {
			return nil, err
		}
		stats = append(stats, p)
	}
	return stats, rows.Err()
}

func (s *Store) Cleanup(retentionDays int) error {
	cutoff := time.Now().Unix() - int64(retentionDays)*86400
	_, err := s.db.Exec(`DELETE FROM access_log WHERE bucket_start < ?`, cutoff)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

var privateNets []*net.IPNet

func init() {
	cidrs := []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1/128", "fc00::/7"}
	for _, cidr := range cidrs {
		_, n, _ := net.ParseCIDR(cidr)
		privateNets = append(privateNets, n)
	}
}

func IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
