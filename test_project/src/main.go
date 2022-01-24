//xim:HandlerModule("./backend")
package main

//goland:noinspection GoRedundantImportAlias
import (
	"fmt"
	"github.com/kongchengpro/xim"
	"github.com/kongchengpro/xim/api/dom"
	. "github.com/kongchengpro/xim/components/button"
	. "github.com/kongchengpro/xim/components/root"
	. "github.com/kongchengpro/xim/components/text"
	. "github.com/kongchengpro/xim/components/view"
	"github.com/kongchengpro/xim/types/callback"
	. "github.com/kongchengpro/xim/types/component"
	backend "main/backend"
)

func main() {
	xim.SetTitle("Hello Xim")
	xim.Init(&View{
		Components: Cs{
			&Text{
				Name:    "MainText",
				Content: "Hello Xim",
				Style: &TextStyle{
					FontSize: "40px",
				},
			},
			&Button{
				Content: "Click Me",
				OnClick: func(this callback.Value, args ...callback.Value) {
					fmt.Printf("%#v\n%#v\n", this, args)
					c, ok := dom.GetComponentByPath("MainText").(*Text)
					if ok {
						go func() {
							c.Content = backend.ConcatStrings("Hello ", "Kogic")
							xim.Refresh(c)
						}()
					}
				},
			},
		},
	}, Root{})
	select {}
}
