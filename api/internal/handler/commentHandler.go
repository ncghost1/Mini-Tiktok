package handler

import (
	"net/http"

	"Mini-Tiktok/api/internal/logic"
	"Mini-Tiktok/api/internal/svc"
	"Mini-Tiktok/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func commentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CommentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCommentLogic(r.Context(), svcCtx)
		resp, err := l.Comment(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
