package favorite

import (
	"net/http"

	"sea-try-go/service/common/response"
	"sea-try-go/service/favorite/api/internal/logic/favorite"
	"sea-try-go/service/favorite/api/internal/svc"
	"sea-try-go/service/favorite/api/internal/types"
	favoritecommon "sea-try-go/service/favorite/common"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetFavoriteFoldersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetFavoriteFoldersReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.OkJsonCtx(r.Context(), w, &response.Response{
				Code: favoritecommon.ErrorInvalidParam,
				Msg:  favoritecommon.GetErrMsg(favoritecommon.ErrorInvalidParam),
				Data: nil,
			})
			return
		}

		l := favorite.NewGetFavoriteFoldersLogic(r.Context(), svcCtx)
		resp, code := l.GetFavoriteFolders(&req)
		httpx.OkJsonCtx(r.Context(), w, &response.Response{
			Code: code,
			Msg:  favoritecommon.GetErrMsg(code),
			Data: resp,
		})
	}
}
