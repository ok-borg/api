package v2

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

	r.GET("/v2/redirect/github/authorize", common.RedirectGithubAuthorize)
	r.POST("/v2/auth/github", common.GithubAuth)

	r.GET("/v2/query", q)

	// authenticated endpoints
	r.GET("/v2/user", access.MaybeAuth(db, common.GetUser))

	// snippets
	r.GET("/v2/p/:id/:owner", access.MaybeAuth(db, getSnippet))
	r.GET("/v2/latest/:owner", access.IfAuth(db, getLatestSnippets))
	r.POST("/v2/p", access.IfAuth(db, access.Control(createSnippet, access.Create)))
	//r.DELETE("/v2/p/:id", access.IfAuth(deleteSnippet))
	r.PUT("/v2/p", access.IfAuth(db, access.Control(updateSnippet, access.Update)))
	r.POST("/v2/worked", access.IfAuth(db, snippetWorked))
	r.POST("/v2/slack", common.SlackCommand)

	// organizations
	r.POST("/v2/organizations", access.IfAuth(db, common.CreateOrganization))
	r.GET("/v2/organizations", access.IfAuth(db, common.ListUserOrganizations))

	// not rest at all but who cares ?
	r.POST("/v2/organizations/leave/:id", access.IfAuth(db, common.LeaveOrganization))
	r.POST("/v2/organizations/expel/:oid/user/id/:uid",
		access.IfAuth(db, common.ExpelUserFromOrganization))
	r.POST("/v2/organizations/admins/:oid/user/id/:uid",
		access.IfAuth(db, common.GrantAdminRightToUser))

	// organizations-join-links
	// this is only allowed for the organization admin
	r.POST("/v2/organization-join-links", access.IfAuth(db, common.CreateOrganizationJoinLink))
	r.DELETE("/v2/organization-join-links/id/:id",
		access.IfAuth(db, common.DeleteOrganizationJoinLink))
	// get a join link for a specific organization
	// this is allowed only by the organization admin in order to share it again, or delete it.
	r.GET("/v2/organization-join-links/organizations/:id",
		access.IfAuth(db, common.GetOrganizationJoinLinkByOrganizationId))
	// get a join link from a join-link id.
	r.GET("/v2/organization-join-links/id/:id",
		access.IfAuth(db, common.GetOrganizationJoinLink))
	// accept join link
	// not restful at all, but pretty to read
	r.POST("/v2/join/:id", access.IfAuth(db, common.JoinOrganization))
}
