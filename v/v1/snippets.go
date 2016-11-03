package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/cihub/seelog"
	httpr "github.com/julienschmidt/httprouter"
	"github.com/ok-borg/api/endpoints"
	"github.com/ok-borg/api/types"
	"github.com/ok-borg/api/v"
)

func q(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	size := 5
	s, err := strconv.ParseInt(r.FormValue("l"), 10, 32)
	if err == nil && s > 0 {
		size = int(s)
	}
	res, err := ep.Query(r.FormValue("q"), size, r.FormValue("p") == "true")
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, err.Error())
	}
	bs, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(bs))
}

func getLatestSnippets(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	res, err := ep.GetLatestSnippets(endpoints.PublicBorgSnippet)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, err.Error())
	}
	bs, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	common.WriteResponse(w, http.StatusOK, string(bs))
}

func createSnippet(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}
	var snipp types.Problem
	if err := json.Unmarshal(body, &snipp); err != nil {
		log.Errorf("Invalid snippet, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid snippet")
		return
	}
	err = ep.CreateSnippet(&snipp, endpoints.PublicBorgSnippet, ctx.Value("userId").(string))
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to unmarshal snippet")
		return
	}
	common.WriteJsonResponse(w, http.StatusOK, snipp)
}

func updateSnippet(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}
	var snipp types.Problem
	if err := json.Unmarshal(body, &snipp); err != nil {
		log.Errorf("[updateSnippet] invalid snippet, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid snippet")
		return
	}
	err = ep.UpdateSnippet(&snipp, endpoints.PublicBorgSnippet, ctx.Value("userId").(string))
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: error")
		return
	}
	common.WriteResponse(w, http.StatusOK, "{}")
}

func snippetWorked(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}
	s := struct {
		Query string
		Id    string
	}{}
	if err := json.Unmarshal(body, &s); err != nil {
		log.Errorf("[updateSnippet] invalid worked request, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid worked request")
		return
	}
	err = ep.Worked(s.Id, s.Query)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: error: "+err.Error())
		return
	}
	common.WriteResponse(w, http.StatusOK, "{}")
}

func getSnippet(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}
	snipp, err := ep.GetSnippet(endpoints.PublicBorgSnippet, id)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: Failed to get snippet")
		return
	}
	if snipp == nil {
		common.WriteResponse(w, http.StatusNotFound, "borg-api: snippet not found")
		return
	}
	bs, _ := json.Marshal(snipp)
	common.WriteResponse(w, http.StatusOK, string(bs))
}
