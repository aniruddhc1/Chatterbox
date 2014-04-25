package handlers

import (
	"github.com/sunfmin/mangotemplate"
	"mango"
	"net/http"
	"html/template"
}

type RenderData struct{
	Username string
}

type Provider struct{
	//intentionally blank
}

type Header struct{
	//intentionally blank
}

func Home(env Env) (status Status, header Headers, body Body){
	mangotemplate.ForRender(env, "templates/startPage.html", nil)
	header = Headers{}
	return
}


func Join (env Env) (status Status, header Headers, body Body){
	username := env.Request().FormValue("username")
	if(username == ""){
		return Redirect(http.StatusFound, "/")
	}

	mangotemplate.ForRender(env, "chats/index", &RenderData{Username : username})
	header := Headers{}
	return
}

func LayoutAndRender() (layout Middleware, render Middleware){
	tpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		panic(err)
	}
	layout = mangotemplate.MakeLayout(tpl, "main", &Provider{})
	render = mangotemplate.MakeRenderer(tpl)
	return
}
