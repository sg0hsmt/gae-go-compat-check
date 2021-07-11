package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"golang.org/x/xerrors"
	"google.golang.org/appengine/v2"
	"google.golang.org/appengine/v2/delay"
	"google.golang.org/appengine/v2/log"
	"google.golang.org/appengine/v2/mail"
	"google.golang.org/appengine/v2/memcache"
	"google.golang.org/appengine/v2/module"
	"google.golang.org/appengine/v2/runtime"
	"google.golang.org/appengine/v2/taskqueue"
	"google.golang.org/appengine/v2/urlfetch"
	"google.golang.org/appengine/v2/user"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.NoCache)

	r.Get("/", indexHandler)
	r.Get("/env", envHandler)
	r.Get("/appengine", appengineHandler)
	r.Get("/delay", delayHandler)
	r.Get("/log", logHandler)
	r.Get("/mail", mailHandler)
	r.Get("/memcache", memcacheHandler)
	r.Get("/module", moduleHandler)
	r.Get("/runtime", runtimeHandler)
	r.Get("/taskqueue", taskqueueHandler)
	r.Get("/urlfetch", urlfetchHandler)
	r.Get("/user", userHandler)

	r.Get("/worker", workerHandler)

	http.Handle("/", r)
	appengine.Main()
}

func isCI() bool {
	_, exist := os.LookupEnv("CI")
	return exist
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "Hello, World!")
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	// List for environment variables.
	// see also https://cloud.google.com/appengine/docs/standard/go111/runtime#environment_variables
	envNames := []string{
		"GAE_APPLICATION",
		"GAE_DEPLOYMENT_ID",
		"GAE_ENV",
		"GAE_INSTANCE",
		"GAE_MEMORY_MB",
		"GAE_RUNTIME",
		"GAE_SERVICE",
		"GAE_VERSION",
		"GOOGLE_CLOUD_PROJECT",
		"PORT",
		"RUN_WITH_DEVAPPSERVER",
		"CI", // Environment variable names often used in CI.
	}

	buf := make([]string, 0, len(envNames))
	for _, name := range envNames {
		buf = append(buf, fmt.Sprintf("%s: %q", name, os.Getenv(name)))
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, strings.Join(buf, "\n"))
}

func appengineHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	module := ""
	version := ""
	instance := ""
	moduleHostName := ""

	if !isCI() {
		// Require metadata server access.
		module = appengine.ModuleName(ctx)
		version = appengine.VersionID(ctx)
		instance = appengine.InstanceID()

		res, err := appengine.ModuleHostname(ctx, "", "", "")
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.PlainText(w, r, xerrors.Errorf("module hostname: %w", err).Error())
			return
		}

		moduleHostName = res
	}

	buf := []string{
		fmt.Sprintf("AppID: %q", appengine.AppID(ctx)),
		fmt.Sprintf("Datacenter: %q", appengine.Datacenter(ctx)),
		fmt.Sprintf("DefaultVersionHostname: %q", appengine.DefaultVersionHostname(ctx)),
		fmt.Sprintf("InstanceID: %q", instance),
		fmt.Sprintf("IsAppEngine: %v", appengine.IsAppEngine()),
		fmt.Sprintf("IsDevAppServer: %v", appengine.IsDevAppServer()),
		fmt.Sprintf("IsFlex: %v", appengine.IsFlex()),
		fmt.Sprintf("IsSecondGen: %v", appengine.IsSecondGen()),
		fmt.Sprintf("IsStandard: %v", appengine.IsStandard()),
		fmt.Sprintf("ModuleHostname: %q", moduleHostName),
		fmt.Sprintf("ModuleName: %q", module),
		fmt.Sprintf("RequestID: %q", appengine.RequestID(ctx)),
		fmt.Sprintf("ServerSoftware: %q", appengine.ServerSoftware()),
		fmt.Sprintf("VersionID: %q", version),
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, strings.Join(buf, "\n"))
}

func delayHandler(w http.ResponseWriter, r *http.Request) {
	fn := delay.Func("delay-function", func(ctx context.Context) {
		log.Infof(ctx, "execute delay-function")
	})

	ctx := r.Context()
	if err := fn.Call(ctx); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("delay call: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "ok")
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Debugf(ctx, "debug log at %s", time.Now().Format(time.RFC3339Nano))
	time.Sleep(time.Millisecond)

	log.Infof(ctx, "info log at %s", time.Now().Format(time.RFC3339Nano))
	time.Sleep(time.Millisecond)

	log.Warningf(ctx, "warning log at %s", time.Now().Format(time.RFC3339Nano))
	time.Sleep(time.Millisecond)

	log.Errorf(ctx, "error log at %s", time.Now().Format(time.RFC3339Nano))
	time.Sleep(time.Millisecond)

	log.Criticalf(ctx, "critical log at %s", time.Now().Format(time.RFC3339Nano))

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "ok")
}

func mailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cur := user.Current(ctx)
	if cur == nil {
		render.Status(r, http.StatusOK)
		render.PlainText(w, r, "user is not login")
		return
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("noreply@%s.appspotmail.com", os.Getenv("GOOGLE_CLOUD_PROJECT")),
		To:      []string{cur.Email},
		Subject: "gae-go-compat test mail",
		Body:    "This is a test mail from GAE/Go Mail API.",
	}
	if err := mail.Send(ctx, msg); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("mail send: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "ok")
}

func memcacheHandler(w http.ResponseWriter, r *http.Request) {
	const key = "memcacheKey"

	ctx := r.Context()

	item, err := memcache.Get(ctx, key)
	if err == nil {
		render.Status(r, http.StatusOK)
		render.PlainText(w, r, fmt.Sprintf("cache hit: %q", string(item.Value)))
		return
	}

	if err != memcache.ErrCacheMiss {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("memcache get: %w", err).Error())
		return
	}

	newItem := &memcache.Item{
		Key:        key,
		Value:      []byte("Hello, Memcache"),
		Expiration: time.Hour,
	}
	if err := memcache.Set(ctx, newItem); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("memcache set: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "cache miss")
}

func moduleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	moduleName, err := module.DefaultVersion(ctx, "")
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("default version: %w", err).Error())
		return
	}

	moduleList, err := module.List(ctx)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("list: %w", err).Error())
		return
	}

	versions, err := module.Versions(ctx, "")
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("versions: %w", err).Error())
		return
	}

	buf := []string{
		fmt.Sprintf("DefaultVersion: %q", moduleName),
		fmt.Sprintf("List: %q", moduleList),
		fmt.Sprintf("Versions: %q", versions),
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, strings.Join(buf, "\n"))
}

func runtimeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stat, err := runtime.Stats(ctx)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("stats: %w", err).Error())
		return
	}

	buf := []string{
		fmt.Sprintf("Statistics.CPU.Total: %f", stat.CPU.Total),
		fmt.Sprintf("Statistics.CPU.Rate1M: %f", stat.CPU.Rate1M),
		fmt.Sprintf("Statistics.CPU.Rate10M: %f", stat.CPU.Rate10M),
		fmt.Sprintf("Statistics.RAM.Current: %f", stat.RAM.Current),
		fmt.Sprintf("Statistics.RAM.Average1M: %f", stat.RAM.Average1M),
		fmt.Sprintf("Statistics.RAM.Average10M: %f", stat.RAM.Average10M),
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, strings.Join(buf, "\n"))
}

func taskqueueHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tasks := make([]*taskqueue.Task, 0, 10)
	for i := 0; i < 10; i++ {
		tasks = append(tasks, &taskqueue.Task{
			Method:  "PULL",
			Payload: []byte(fmt.Sprintf("Hello, PullTask %d", i)),
		})
	}

	if _, err := taskqueue.AddMulti(ctx, tasks, "pull-queue"); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("add multi: %w", err).Error())
		return
	}

	task := &taskqueue.Task{
		Path:   "/worker",
		Method: http.MethodGet,
	}
	if _, err := taskqueue.Add(ctx, task, ""); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("add: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "ok")
}

func urlfetchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client := urlfetch.Client(ctx)
	resp, err := client.Get("https://httpbin.org/get")
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("get: %w", err).Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("read all: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, string(body))
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cur := user.Current(ctx)
	if cur == nil {
		render.Status(r, http.StatusOK)
		render.PlainText(w, r, "user is not login")
		return
	}

	buf := []string{
		fmt.Sprintf("IsAdmin: %v", user.IsAdmin(ctx)),
		fmt.Sprintf("User.Email: %q", cur.Email),
		fmt.Sprintf("User.AuthDomain: %q", cur.AuthDomain),
		fmt.Sprintf("User.Admin: %v", cur.Admin),
		fmt.Sprintf("User.ID: %q", cur.ID),
		fmt.Sprintf("User.ClientID: %q", cur.ClientID),
		fmt.Sprintf("User.FederatedIdentity: %q", cur.FederatedIdentity),
		fmt.Sprintf("User.FederatedProvider: %q", cur.FederatedProvider),
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, strings.Join(buf, "\n"))
}

func workerHandler(w http.ResponseWriter, r *http.Request) {
	// Check for task queue headers.
	// see also https://cloud.google.com/appengine/docs/standard/go111/taskqueue/push/creating-handlers#reading_request_headers
	if r.Header.Get("X-Appengine-QueueName") == "" {
		render.Status(r, http.StatusForbidden)
		render.PlainText(w, r, "require queue name header")
		return
	}

	ctx := r.Context()

	tasks, err := taskqueue.Lease(ctx, 1000, "pull-queue", 60)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("lease: %w", err).Error())
		return
	}

	log.Infof(ctx, fmt.Sprintf("pull %d tasks", len(tasks)))

	if err := taskqueue.DeleteMulti(ctx, tasks, "pull-queue"); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, xerrors.Errorf("delete multi: %w", err).Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "ok")
}
