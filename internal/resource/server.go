package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// Server 暴露桥的 HTTP 接口：来源框架或插件可请求“把某资源共享出来”。
// 仅接受 file_id + kind，返回清洗后的 ResourceRef；不接收也不返回任何隐私数据。
type Server struct {
	bridge *Bridge
}

// NewServer 创建 HTTP 服务
func NewServer(bridge *Bridge) *Server {
	return &Server{bridge: bridge}
}

type ingestRequest struct {
	Kind   string `json:"kind"`
	FileID string `json:"file_id"`
}

// Handler 返回 http.Handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ingest", s.handleIngest)
	mux.HandleFunc("/index", s.handleIndex)
	return mux
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持 POST"})
		return
	}
	var req ingestRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求体解析失败"})
		return
	}
	if req.FileID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 file_id"})
		return
	}
	kind := model.ResourceKind(req.Kind)
	if kind == "" {
		kind = model.ResourceKindFile
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	ref, err := s.bridge.Fetch(ctx, FetchRequest{Kind: kind, FileID: req.FileID})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ref)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	idx, err := s.bridge.store.LoadIndex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, idx)
}

// Serve 在指定地址监听
func (s *Server) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("桥服务监听失败: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
