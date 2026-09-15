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

type MapPoint struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Success     int64   `json:"success"`
	Rejected    int64   `json:"rejected"`
	Blocked     int64   `json:"blocked"`
}

type TimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Success   int64     `json:"success"`
	Rejected  int64     `json:"rejected"`
	Blocked   int64     `json:"blocked"`
}

type PathStat struct {
	Path     string `json:"path"`
	Method   string `json:"method"`
	Hostname string `json:"hostname"`
	Count    int64  `json:"count"`
	Rejected int64  `json:"rejected"`
	Blocked  bool   `json:"blocked"`
}

type IPStat struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Success     int64   `json:"success"`
	Rejected    int64   `json:"rejected"`
	Blocked     int64   `json:"blocked"`
}

// classification values stored in access_log.classification
const (
	clsPending  = 0 // auth redirect, awaiting resolution
	clsSuccess  = 1
	clsRejected = 2 // 401 or 403 — auth/authorization failure
	clsBlocked  = 3
	clsInternal = 4 // auth callback (/login/callback) — excluded from stats
)

type Store struct {
	db   sqlite.Database
	stop chan struct{}
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sqlite.OpenDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, stop: make(chan struct{})}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS access_log (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp      INTEGER NOT NULL,
			hostname       TEXT    NOT NULL,
			client_ip      TEXT    NOT NULL,
			method         TEXT    NOT NULL DEFAULT '',
			path           TEXT    NOT NULL DEFAULT '',
			status_code    INTEGER NOT NULL DEFAULT 0,
			duration_ms    INTEGER NOT NULL DEFAULT 0,
			classification INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_ts       ON access_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_hostname ON access_log(hostname)`,
		`CREATE INDEX IF NOT EXISTS idx_access_log_ip_ts    ON access_log(client_ip, hostname, timestamp)`,
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

	// Normalise local network geo entries once.
	if _, err := s.db.Exec(`UPDATE geo_cache SET lat = 30, lon = -40, country = 'Local Network', country_code = 'LC', city = 'Local'
		WHERE country IN ('Local Network', 'Local') OR city IN ('Local', 'Localhost')`); err != nil {
		return err
	}

	return nil
}

// StartBackgroundCleanup runs the pending-timeout and retention cleanup loops.
// Call it once after NewStore; stop by closing the Store.
func (s *Store) StartBackgroundCleanup(retentionDays int) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Age out auth redirects that were never followed up.
				cutoff := time.Now().Add(-10 * time.Minute).Unix()
				_, _ = s.db.Exec(
					`UPDATE access_log SET classification=? WHERE classification=? AND timestamp<?`,
					clsBlocked, clsPending, cutoff,
				)
			case <-s.stop:
				// Final retention cleanup.
				cutoff := time.Now().Unix() - int64(retentionDays)*86400
				_, _ = s.db.Exec(`DELETE FROM access_log WHERE timestamp<?`, cutoff)
				return
			}
		}
	}()
}

// RecordRequest writes a single request to the store and resolves pending auth records
// when an auth callback succeeds.
func (s *Store) RecordRequest(ts time.Time, hostname, clientIP, method, path string, statusCode int, durationMs int64, classification string) error {
	cls := s.classify(path, statusCode, classification)
	_, err := s.db.Exec(
		`INSERT INTO access_log (timestamp, hostname, client_ip, method, path, status_code, duration_ms, classification) VALUES (?,?,?,?,?,?,?,?)`,
		ts.Unix(), hostname, clientIP, method, path, statusCode, durationMs, cls,
	)
	if err != nil {
		return err
	}
	if cls == clsInternal {
		s.resolveAuthPending(clientIP, hostname, ts)
	}
	return nil
}

func (s *Store) classify(path string, statusCode int, classification string) int {
	switch classification {
	case "blocked":
		return clsBlocked
	case "auth_redirect":
		return clsPending
	}
	if path == "/login/callback" && statusCode == 302 {
		return clsInternal
	}
	if statusCode == 401 || statusCode == 403 {
		return clsRejected
	}
	return clsSuccess
}

func (s *Store) resolveAuthPending(clientIP, hostname string, callbackTime time.Time) {
	window := callbackTime.Add(-10 * time.Minute).Unix()
	_, _ = s.db.Exec(
		`UPDATE access_log SET classification=? WHERE client_ip=? AND hostname=? AND classification=? AND timestamp>=? AND timestamp<=?`,
		clsSuccess, clientIP, hostname, clsPending, window, callbackTime.Unix(),
	)
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
	query := `SELECT g.lat, g.lon, g.country, g.country_code, g.city,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS success,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS errors,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS blocked
	FROM access_log a
	JOIN geo_cache g ON a.client_ip = g.ip
	WHERE a.timestamp >= ? AND a.timestamp <= ?
	  AND a.classification NOT IN (?,?)`
	args := []any{clsSuccess, clsRejected, clsBlocked, from.Unix(), to.Unix(), clsPending, clsInternal}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	if clientIP != "" {
		query += " AND a.client_ip = ?"
		args = append(args, clientIP)
	}
	query += " GROUP BY g.lat, g.lon, g.country, g.country_code, g.city ORDER BY COUNT(*) DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []MapPoint
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(&p.Lat, &p.Lon, &p.Country, &p.CountryCode, &p.City, &p.Success, &p.Rejected, &p.Blocked); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (s *Store) GetTimeSeries(from, to time.Time, hostname, granularity, clientIP, city, country string) ([]TimePoint, error) {
	var bucketSeconds int64 = 3600
	switch granularity {
	case "day":
		bucketSeconds = 86400
	case "week":
		bucketSeconds = 604800
	}
	hasLocation := city != "" && country != ""

	query := `SELECT (a.timestamp / ?) * ? as ts,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS blocked,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS errors,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS success
	FROM access_log a`
	if hasLocation {
		query += ` JOIN geo_cache g ON a.client_ip = g.ip`
	}
	query += ` WHERE a.timestamp >= ? AND a.timestamp <= ? AND a.classification NOT IN (?,?)`
	args := []any{bucketSeconds, bucketSeconds, clsBlocked, clsRejected, clsSuccess, from.Unix(), to.Unix(), clsPending, clsInternal}

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
		if err := rows.Scan(&ts, &p.Blocked, &p.Rejected, &p.Success); err != nil {
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

	filteredFrom := "FROM access_log a"
	if hasLocation {
		filteredFrom += " JOIN geo_cache g ON a.client_ip = g.ip"
	}
	filteredFrom += " WHERE a.timestamp >= ? AND a.timestamp <= ?"
	args := []any{from.Unix(), to.Unix()}

	if hostname != "" {
		filteredFrom += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	if clientIP != "" {
		filteredFrom += " AND a.client_ip = ?"
		args = append(args, clientIP)
	}
	if hasLocation {
		filteredFrom += " AND g.city = ? AND g.country = ?"
		args = append(args, city, country)
	}

	query := `WITH filtered AS (
		SELECT a.path, a.method, a.hostname, a.classification
		` + filteredFrom + `
	)
	SELECT path, method, hostname, total, errors, 0
	FROM (
		SELECT f.path, f.method, f.hostname,
		       COUNT(*) AS total,
		       COUNT(CASE WHEN f.classification=? THEN 1 END) AS errors
		FROM filtered f
		WHERE f.classification IN (?,?) AND f.path != ''
		GROUP BY f.path, f.method, f.hostname
		ORDER BY total DESC LIMIT ?
	)
	UNION ALL
	SELECT '' AS path, '' AS method, f.hostname, COUNT(*) AS total, 0, 1
	FROM filtered f
	WHERE f.classification = ?
	GROUP BY f.hostname`
	args = append(args, clsRejected, clsSuccess, clsRejected, limit, clsBlocked)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PathStat
	for rows.Next() {
		var p PathStat
		var isBlocked int
		if err := rows.Scan(&p.Path, &p.Method, &p.Hostname, &p.Count, &p.Rejected, &isBlocked); err != nil {
			return nil, err
		}
		p.Blocked = isBlocked == 1
		stats = append(stats, p)
	}
	return stats, rows.Err()
}

func (s *Store) GetIPStats(from, to time.Time, hostname string, limit int) ([]IPStat, error) {
	query := `SELECT a.client_ip,
		COALESCE(g.country,''), COALESCE(g.country_code,''), COALESCE(g.city,''),
		COALESCE(g.lat,0), COALESCE(g.lon,0),
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS blocked,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS errors,
		COUNT(CASE WHEN a.classification=? THEN 1 END) AS success
	FROM access_log a
	LEFT JOIN geo_cache g ON a.client_ip = g.ip
	WHERE a.timestamp >= ? AND a.timestamp <= ?
	  AND a.classification NOT IN (?,?)`
	args := []any{clsBlocked, clsRejected, clsSuccess, from.Unix(), to.Unix(), clsPending, clsInternal}

	if hostname != "" {
		query += " AND a.hostname = ?"
		args = append(args, hostname)
	}
	query += " GROUP BY a.client_ip ORDER BY COUNT(*) DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []IPStat
	for rows.Next() {
		var p IPStat
		if err := rows.Scan(&p.IP, &p.Country, &p.CountryCode, &p.City, &p.Lat, &p.Lon, &p.Blocked, &p.Rejected, &p.Success); err != nil {
			return nil, err
		}
		stats = append(stats, p)
	}
	return stats, rows.Err()
}

func (s *Store) Close() error {
	close(s.stop)
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
