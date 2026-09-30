package controllers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	beego "github.com/beego/beego/v2/server/web"
)

func TestMain(m *testing.M) {
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(thisFile))

	if err := os.Chdir(projectRoot); err != nil {
		panic("could not move to the project root: " + err.Error())
	}

	beego.BConfig.WebConfig.ViewsPath = "views"
	if err := beego.AddViewPath("views"); err != nil {
		panic("could not load the templates: " + err.Error())
	}

	os.Exit(m.Run())
}

func serve(t *testing.T, app *beego.HttpServer, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	app.Handlers.ServeHTTP(rec, req)
	return rec
}

func serveWithReferer(t *testing.T, app *beego.HttpServer, method, target, referer string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Referer", referer)
	rec := httptest.NewRecorder()
	app.Handlers.ServeHTTP(rec, req)
	return rec
}

func newApp() *beego.HttpServer {
	app := beego.NewHttpSever()
	app.Cfg.WebConfig.ViewsPath = "views"
	return app
}
