package api

import (
	"net/http/httptest"
	"testing"
)

func TestResolveConfigTarget(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback string
		want     string
		wantErr  bool
	}{
		{name: "默认 clash meta", want: "clash-meta"},
		{name: "显式 clash", raw: "clash", want: "clash-meta"},
		{name: "显式 clash_meta", raw: "clash_meta", want: "clash-meta"},
		{name: "显式 stash", raw: "stash", want: "stash"},
		{name: "显式 surge", raw: "surge", want: "surge"},
		{name: "显式 sing_box", raw: "sing_box", want: "sing-box"},
		{name: "使用模板 fallback", fallback: "sing_box", want: "sing-box"},
		{name: "默认 stash fallback", fallback: "stash", want: "stash"},
		{name: "非法 target", raw: "loon", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveConfigTarget(tt.raw, tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望返回错误，实际成功: %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveConfigTarget() 返回错误: %v", err)
			}
			if got != tt.want {
				t.Fatalf("期望 target=%q，实际为 %q", tt.want, got)
			}
		})
	}
}

func TestParseTemplateLookup(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		wantName  string
		wantID    int
		wantErr   bool
	}{
		{name: "template 名称", targetURL: "/convert?template=my-template", wantName: "my-template"},
		{name: "template 数字视为 id", targetURL: "/convert?template=12", wantID: 12},
		{name: "template_id 优先", targetURL: "/convert?template=my-template&template_id=8", wantID: 8},
		{name: "template_name", targetURL: "/convert?template_name=tpl-a", wantName: "tpl-a"},
		{name: "非法 template_id", targetURL: "/convert?template_id=abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.targetURL, nil)
			gotName, gotID, err := parseTemplateLookup(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("期望返回错误，实际成功")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTemplateLookup() 返回错误: %v", err)
			}
			if gotName != tt.wantName || gotID != tt.wantID {
				t.Fatalf("期望 name=%q id=%d，实际为 name=%q id=%d", tt.wantName, tt.wantID, gotName, gotID)
			}
		})
	}
}

func TestParseConvertRequestParams(t *testing.T) {
	tests := []struct {
		name           string
		targetURL      string
		wantSubURL     string
		wantTemplate   string
		wantTemplateID int
		wantTarget     string
		wantErr        bool
	}{
		{
			name:         "支持未编码订阅地址",
			targetURL:    "/convert?url=https://example.com/sub?target=clash&filename=demo&emoji=true&url=https%3A%2F%2Forigin.example.com%2Fsub%3Ftoken%3Dabc&template=clash",
			wantSubURL:   "https://example.com/sub?target=clash&filename=demo&emoji=true&url=https%3A%2F%2Forigin.example.com%2Fsub%3Ftoken%3Dabc",
			wantTemplate: "clash",
		},
		{
			name:           "支持编码订阅地址和 target",
			targetURL:      "/convert?target=surge&url=https%3A%2F%2Fexample.com%2Fsub%3Fa%3D1%26b%3D2&template_id=9",
			wantSubURL:     "https://example.com/sub?a=1&b=2",
			wantTemplateID: 9,
			wantTarget:     "surge",
		},
		{
			name:      "非法 template_id",
			targetURL: "/convert?url=https://example.com/sub&template_id=abc",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.targetURL, nil)
			got, err := parseConvertRequestParams(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("期望返回错误，实际成功")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseConvertRequestParams() 返回错误: %v", err)
			}
			if got.subURL != tt.wantSubURL || got.templateRef != tt.wantTemplate || got.templateID != tt.wantTemplateID || got.target != tt.wantTarget {
				t.Fatalf("期望 subURL=%q template=%q templateID=%d target=%q，实际为 subURL=%q template=%q templateID=%d target=%q",
					tt.wantSubURL, tt.wantTemplate, tt.wantTemplateID, tt.wantTarget,
					got.subURL, got.templateRef, got.templateID, got.target)
			}
		})
	}
}

func TestConfigResponseMetaForTarget(t *testing.T) {
	tests := []struct {
		target          string
		wantFilename    string
		wantContentType string
	}{
		{target: "clash-meta", wantFilename: "clash_meta_config.yaml", wantContentType: "text/yaml; charset=utf-8"},
		{target: "stash", wantFilename: "stash_config.yaml", wantContentType: "text/yaml; charset=utf-8"},
		{target: "surge", wantFilename: "surge_config.conf", wantContentType: "text/plain; charset=utf-8"},
		{target: "sing-box", wantFilename: "sing_box_config.json", wantContentType: "application/json; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := configResponseMetaForTarget(tt.target)
			if got.filename != tt.wantFilename || got.contentType != tt.wantContentType {
				t.Fatalf("target=%s 时，期望 filename=%q contentType=%q，实际为 filename=%q contentType=%q",
					tt.target, tt.wantFilename, tt.wantContentType, got.filename, got.contentType)
			}
		})
	}
}

func TestBuildConvertLogToken(t *testing.T) {
	tests := []struct {
		name   string
		params convertRequestParams
		want   string
	}{
		{name: "默认", params: convertRequestParams{}, want: "convert"},
		{name: "按模板名", params: convertRequestParams{templateRef: "clash"}, want: "convert:template=clash"},
		{name: "按模板ID", params: convertRequestParams{templateID: 7}, want: "convert:template_id=7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildConvertLogToken(tt.params); got != tt.want {
				t.Fatalf("期望 token=%q，实际为 %q", tt.want, got)
			}
		})
	}
}
