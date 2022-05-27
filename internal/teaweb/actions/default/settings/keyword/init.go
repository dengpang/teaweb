package keyword

import (
	"github.com/TeaWeb/build/internal/teaweb/actions/default/settings"
	"github.com/TeaWeb/build/internal/teaweb/configs"
	"github.com/TeaWeb/build/internal/teaweb/helpers"
	"github.com/iwind/TeaGo"
)

func init() {
	// 路由设置
	TeaGo.BeforeStart(func(server *TeaGo.Server) {
		server.
			Helper(&helpers.UserMustAuth{
				Grant: configs.AdminGrantAll,
			}).
			Helper(new(settings.Helper)).
			Prefix("/settings/keyword").
			Get("", new(IndexAction)).
			GetPost("/update", new(UpdateAction)).
			//Post("/delete", new(DeleteAction)).
			//Post("/restore", new(RestoreAction)).
			//Get("/download", new(DownloadAction)).
			//Post("/clean", new(CleanAction)).
			EndAll()
	})

}
