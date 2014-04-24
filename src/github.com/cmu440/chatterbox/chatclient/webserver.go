//package main
//
//import "github.com/go-martini/martini"
//import "github.com/martini-contrib-master/render"
//
//func main() {
//m := martini.Classic()
//	m.Get("/", func() string {
//			return "Hello world!"
//		})
//
//	opts := &render.Options{
//		Directory: "github.com/cmu440/chatterbox/templates",
//	}
//
//	m.Use(render.Renderer(*opts))
//
//	m.Get("/index.html", func(r render.Render){
//			r.HTML(200, "index", nil)
//		})
//
//	m.Use(martini.Static("github.com/cmu440/chatterbox/templates"))
//
//	m.Run()
//}

//package main

package chatclient
