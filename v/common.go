package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/jinzhu/gorm"
	"github.com/jpillora/go-ogle-analytics"
	httpr "github.com/julienschmidt/httprouter"
	"github.com/ok-borg/api/ctxext"
	"github.com/ok-borg/api/endpoints"
	"gopkg.in/olivere/elastic.v3"
)

var (
	client          *elastic.Client
	analyticsClient *ga.Client
	ep              *endpoints.Endpoints
	db              *gorm.DB
	githubClientId  string
)

func Init(
	client_ *elastic.Client,
	analyticsClient_ *ga.Client,
	ep_ *endpoints.Endpoints,
	db_ *gorm.DB,
	githubClientId_ string,
) {
	client = client_
	analyticsClient = analyticsClient_
	ep = ep_
	db = db_
	githubClientId = githubClientId_
}

func WriteJsonResponse(w http.ResponseWriter, status int, body interface{}) {
	rawBody, _ := json.Marshal(body)
	WriteResponse(w, status, string(rawBody))
}

func WriteResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(body)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `%v`, body)
}

func ReadJsonBody(r *http.Request, expectedBody interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.New("unable to read body")
	}
	if err := json.Unmarshal(body, expectedBody); err != nil {
		log.Errorf(
			"[ReadJsonBody] invalid request, %s, input was %s",
			err.Error(), string(body))
		return errors.New("invalid json body format")
	}
	return nil
}

// just redirect the user with the url to the github oauth login with the client_id
// setted in the backend
func RedirectGithubAuthorize(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=read:org",
		githubClientId,
	)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func GithubAuth(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	user, token, err := ep.GithubAuth(string(body))
	if err != nil {
		fmt.Fprintln(w, fmt.Sprintf("Auth failed: %v", err))
		return
	}
	ret := map[string]interface{}{}
	ret["user"] = user
	ret["token"] = token
	bs, err := json.Marshal(ret)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(bs))
}

func GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	user, _ := ctxext.User(ctx)
	bs, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(bs))
}

func CreateOrganization(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {

	// first unmarshal body
	expectedBody := struct{ Name string }{}
	if err := ReadJsonBody(r, &expectedBody); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			fmt.Sprintf("borg-api: %s", err.Error()))
		return
	}

	u, _ := ctxext.User(ctx)
	// lets create an org
	o, err := ep.CreateOrganization(db, u.Id, expectedBody.Name)
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: create organization error: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusOK, o)
}

// create a new organization join link.
// only an administrator of an organization can execute this action
func CreateOrganizationJoinLink(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {

	// first unmarshal body
	expectedBody := struct {
		OrganizationId string
		Ttl            int64
	}{}
	if err := ReadJsonBody(r, &expectedBody); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			fmt.Sprintf("borg-api: %s", err.Error()))
		return
	}

	// check mandatory fields
	if expectedBody.OrganizationId == "" || expectedBody.Ttl <= 0 {
		log.Errorf(
			"[createOrganizationJoinLink] invalid createOrganizationjoinlink body")
		WriteResponse(w, http.StatusBadRequest, "borg-api: invalid body")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	o, err := ep.CreateOrganizationJoinLink(db, u.Id, expectedBody.OrganizationId, expectedBody.Ttl)
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: create organization join link error: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusOK, o)
}

// delete an existing link
// same as previously
func DeleteOrganizationJoinLink(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// delete the organizartion Join Link
	if err := ep.DeleteOrganizationJoinLink(db, u.Id, id); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: delete organization join link error: "+err.Error())
		return
	}
	WriteResponse(w, http.StatusOK, "")
}

// by id
// get an existing link in order to consult the time left for the
// join link, or delete it, or get the the organizastion link for invited
// users to display orgs infos
func GetOrganizationJoinLink(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}

	// get the organization join link
	// no need of user id or anythin
	ojl, err := ep.GetOrganizationJoinLink(db, id)
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: get organization join link error: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusOK, ojl)
}

// get join link for a given organization
// will work only for an admin in order to manage this join-link
func GetOrganizationJoinLinkByOrganizationId(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	ojl, err := ep.GetOrganizationJoinLinkForOrganization(db, u.Id, id)
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: get organization join link error: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusOK, ojl)
}

// join an organization.
// if join link is not expired.
func JoinOrganization(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	if err := ep.JoinOrganization(db, u.Id, id); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: cannot join organization: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusNoContent, "")
}

// list user organization
func ListUserOrganizations(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params) {
	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	orgz, err := ep.ListUserOrganizations(db, u.Id)
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: list user organizations error: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusOK, orgz)
}

// leave an organization,
// you cannot leave an organization if you are the only admin for it
func LeaveOrganization(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params,
) {
	organizationId := p.ByName("id")
	if len(organizationId) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	if err := ep.LeaveOrganization(db, u.Id, organizationId); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: cannot leave organization: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusNoContent, "")
}

// expel an user from an organization,
// you can only do this if you are admin of the organization from
// where you want to expel someone
func ExpelUserFromOrganization(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params,
) {
	organizationId := p.ByName("oid")
	if len(organizationId) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing organizationId url parameter")
		return
	}
	userId := p.ByName("uid")
	if len(userId) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing userId url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	if err := ep.ExpelUserFromOrganization(db, u.Id, userId, organizationId); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: cannot expel from organization: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusNoContent, "")
}

func GrantAdminRightToUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	p httpr.Params,
) {
	organizationId := p.ByName("oid")
	if len(organizationId) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing organizationId url parameter")
		return
	}
	userId := p.ByName("uid")
	if len(userId) == 0 {
		WriteResponse(w, http.StatusBadRequest, "borg-api: Missing userId url parameter")
		return
	}

	u, _ := ctxext.User(ctx)
	// ceate the organizartion Join Link
	if err := ep.GrantAdminRightToUser(db, u.Id, userId, organizationId); err != nil {
		WriteResponse(w, http.StatusInternalServerError,
			"borg-api: cannot expel from organization: "+err.Error())
		return
	}

	WriteJsonResponse(w, http.StatusNoContent, "")
}

func SlackCommand(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	if err := r.ParseForm(); err != nil {
		WriteResponse(w, http.StatusInternalServerError, "Something wrong happened, please try again later.")
		return
	}
	res, err := ep.Slack(r.FormValue("text"))
	if err != nil {
		WriteResponse(w, http.StatusInternalServerError, "Something wrong happened, please try again later.")
		return
	}

	WriteResponse(w, http.StatusOK, res)
}
