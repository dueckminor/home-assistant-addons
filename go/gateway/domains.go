package gateway

import "github.com/gin-gonic/gin"

type Domains struct {
	config *Config
}

func (d *Domains) setupEndpoints(r *gin.RouterGroup) {
	domainsGroup := r.Group("/domains")
	domainsGroup.GET("", d.GET_Domains)
	domainsGroup.POST(":domain", d.POST_Domains)
	domainsGroup.DELETE(":domain", d.DELETE_Domains)
}

func (d *Domains) GET_Domains(c *gin.Context) {
	c.JSON(200, gin.H{"domains": d.config.Domains})
}

func (d *Domains) POST_Domains(c *gin.Context) {
	domain := c.Param("domain")

	// check if domain already exists
	for _, d := range d.config.Domains {
		if d == domain {
			c.JSON(400, gin.H{"error": "domain already exists"})
			return
		}
	}

	d.config.Domains = append(d.config.Domains, domain)
	d.config.save()

	c.JSON(200, gin.H{"domains": d.config.Domains})
}

func (d *Domains) DELETE_Domains(c *gin.Context) {
	domain := c.Param("domain")

	// check if domain exists
	found := false
	for i, existingDomain := range d.config.Domains {
		if existingDomain == domain {
			// remove domain
			d.config.Domains = append(d.config.Domains[:i], d.config.Domains[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(404, gin.H{"error": "domain not found"})
		return
	}

	d.config.save()

	c.JSON(200, gin.H{"domains": d.config.Domains})
}
