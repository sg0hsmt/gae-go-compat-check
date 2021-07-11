package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/appengine/v2/aetest"
	"google.golang.org/appengine/v2/user"
)

func TestHandlers(t *testing.T) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	if err := os.Setenv("CI", "true"); err != nil {
		t.Fatal(err)
	}

	loginUser := &user.User{
		Email:             "mail@example.com",
		AuthDomain:        "",
		Admin:             false,
		ID:                "id",
		ClientID:          "",
		FederatedIdentity: "federated_identity",
		FederatedProvider: "federated_provider",
	}

	check(t, inst, nil, indexHandler, "Hello, World!")

	check(t, inst, nil, appengineHandler, strings.Join([]string{
		`AppID: "testapp"`,
		`Datacenter: ""`,
		`DefaultVersionHostname: ""`,
		`InstanceID: ""`,
		`IsAppEngine: false`,
		`IsDevAppServer: false`,
		`IsFlex: false`,
		`IsSecondGen: false`,
		`IsStandard: false`,
		`ModuleHostname: ""`,
		`ModuleName: ""`,
		`RequestID: ""`,
		`ServerSoftware: "Google App Engine/1.x.x"`,
		`VersionID: ""`,
	}, "\n"))

	check(t, inst, nil, envHandler, strings.Join([]string{
		`GAE_APPLICATION: ""`,
		`GAE_DEPLOYMENT_ID: ""`,
		`GAE_ENV: ""`,
		`GAE_INSTANCE: ""`,
		`GAE_MEMORY_MB: ""`,
		`GAE_RUNTIME: ""`,
		`GAE_SERVICE: ""`,
		`GAE_VERSION: ""`,
		`GOOGLE_CLOUD_PROJECT: ""`,
		`PORT: ""`,
		`RUN_WITH_DEVAPPSERVER: ""`,
		`CI: "true"`,
	}, "\n"))

	check(t, inst, nil, delayHandler, "ok")

	check(t, inst, nil, logHandler, "ok")

	check(t, inst, nil, mailHandler, "user is not login")
	check(t, inst, loginUser, mailHandler, "ok")

	check(t, inst, nil, memcacheHandler, "cache miss")
	check(t, inst, nil, memcacheHandler, "cache hit: \"Hello, Memcache\"")

	// Skip module test.
	// require metadata server access.
	/*
		check(t, inst, nil, moduleHandler, "ok")
	*/

	check(t, inst, nil, runtimeHandler, strings.Join([]string{
		`Statistics.CPU.Total: 0.000000`,
		`Statistics.CPU.Rate1M: 0.000000`,
		`Statistics.CPU.Rate10M: 0.000000`,
		`Statistics.RAM.Current: 0.000000`,
		`Statistics.RAM.Average1M: 0.000000`,
		`Statistics.RAM.Average10M: 0.000000`,
	}, "\n"))

	// Skip task queue test.
	// see also https://github.com/golang/appengine/issues/133
	/*
		check(t, inst, nil, taskqueueHandler, "ok")
		check(t, inst, nil, workerHandler, "ok")
	*/

	check(t, inst, nil, urlfetchHandler, "")

	check(t, inst, nil, userHandler, "user is not login")
	check(t, inst, loginUser, userHandler, strings.Join([]string{
		`IsAdmin: false`,
		`User.Email: "mail@example.com"`,
		`User.AuthDomain: ""`,
		`User.Admin: false`,
		`User.ID: "id"`,
		`User.ClientID: ""`,
		`User.FederatedIdentity: "federated_identity"`,
		`User.FederatedProvider: "federated_provider"`,
	}, "\n"))
}

func check(t *testing.T, inst aetest.Instance, user *user.User, target http.HandlerFunc, body string) {
	t.Helper()

	req, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	if user != nil {
		aetest.Login(user, req)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(target)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status mismatch, want %v, got %v", http.StatusOK, rr.Code)
		return
	}

	if body == "" {
		return
	}

	if diff := cmp.Diff(body, rr.Body.String()); diff != "" {
		t.Errorf("body mismatch (-want +got):\n%s", diff)
	}
}
