package v1

import (
	"github.com/jinzhu/gorm"
	"github.com/jpillora/go-ogle-analytics"
	httpr "github.com/julienschmidt/httprouter"
	"github.com/ok-borg/api/access"
	"github.com/ok-borg/api/endpoints"
	"github.com/ok-borg/api/v"
	"gopkg.in/olivere/elastic.v3"
)

var (
	client          *elastic.Client
	analyticsClient *ga.Client
	ep              *endpoints.Endpoints
	db              *gorm.DB
)

func Init(
	r *httpr.Router,
	client_ *elastic.Client,
	analyticsClient_ *ga.Client,
	ep_ *endpoints.Endpoints,
	db_ *gorm.DB,
) {
	client = client_
	analyticsClient = analyticsClient_
	ep = ep_
	db = db_

	r.GET("/v1/redirect/github/authorize", common.RedirectGithubAuthorize)
	r.POST("/v1/auth/github", common.GithubAuth)

	r.GET("/v1/query", q)

	// authenticated endpoints
	r.GET("/v1/user", access.IfAuth(db, common.GetUser))

	// snippets
	r.GET("/v1/p/:id", getSnippet)
	r.GET("/v1/latest", getLatestSnippets)
	r.POST("/v1/p", access.IfAuth(db, access.Control(createSnippet, access.Create)))
	//r.DELETE("/v1/p/:id", access.IfAuth(deleteSnippet))
	r.PUT("/v1/p", access.IfAuth(db, access.Control(updateSnippet, access.Update)))
	r.POST("/v1/worked", access.IfAuth(db, snippetWorked))
	r.POST("/v1/slack", common.SlackCommand)

	// organizations
	r.POST("/v1/organizations", access.IfAuth(db, common.CreateOrganization))
	r.GET("/v1/organizations", access.IfAuth(db, common.ListUserOrganizations))

	// not rest at all but who cares ?
	r.POST("/v1/organizations/leave/:id", access.IfAuth(db, common.LeaveOrganization))
	r.POST("/v1/organizations/expel/:oid/user/id/:uid",
		access.IfAuth(db, common.ExpelUserFromOrganization))
	r.POST("/v1/organizations/admins/:oid/user/id/:uid",
		access.IfAuth(db, common.GrantAdminRightToUser))

	// organizations-join-links
	// this is only allowed for the organization admin
	r.POST("/v1/organization-join-links", access.IfAuth(db, common.CreateOrganizationJoinLink))
	r.DELETE("/v1/organization-join-links/id/:id",
		access.IfAuth(db, common.DeleteOrganizationJoinLink))
	// get a join link for a specific organization
	// this is allowed only by the organization admin in order to share it again, or delete it.
	r.GET("/v1/organization-join-links/organizations/:id",
		access.IfAuth(db, common.GetOrganizationJoinLinkByOrganizationId))
	// get a join link from a join-link id.
	r.GET("/v1/organization-join-links/id/:id",
		access.IfAuth(db, common.GetOrganizationJoinLink))
	// accept join link
	// not restful at all, but pretty to read
	r.POST("/v1/join/:id", access.IfAuth(db, common.JoinOrganization))
}
