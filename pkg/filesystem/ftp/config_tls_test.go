package ftp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/octohelm/unifs/pkg/strfmt"
	. "github.com/octohelm/x/testing/v2"
)

func TestConfigTLSVersions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		query    string
		min, max uint16
		wantErr  string
	}{
		{name: "未配置时保留默认值"},
		{name: "固定 TLS 1.2", query: "tlsMinVersion=1.2&tlsMaxVersion=1.2", min: tls.VersionTLS12, max: tls.VersionTLS12},
		{name: "只设置下限", query: "tlsMinVersion=1.3", min: tls.VersionTLS13},
		{name: "只设置上限", query: "tlsMaxVersion=1.2", max: tls.VersionTLS12},
		{name: "允许较早版本", query: "tlsMinVersion=1.0&tlsMaxVersion=1.1", min: tls.VersionTLS10, max: tls.VersionTLS11},
		{name: "不限制为已知版本", query: "tlsMinVersion=1.4&tlsMaxVersion=1.10", min: 0x0305, max: 0x030b},
		{name: "非法下限", query: "tlsMinVersion=TLS1.2", wantErr: "tlsMinVersion.*格式"},
		{name: "非法上限", query: "tlsMaxVersion=1.2.0", wantErr: "tlsMaxVersion.*格式"},
		{name: "空值", query: "tlsMinVersion=", wantErr: "tlsMinVersion.*格式"},
		{name: "负数", query: "tlsMinVersion=1.-1", wantErr: "tlsMinVersion.*格式"},
		{name: "非数字", query: "tlsMinVersion=1.x", wantErr: "tlsMinVersion.*格式"},
		{name: "缺少次版本号", query: "tlsMaxVersion=1.", wantErr: "tlsMaxVersion.*格式"},
		{name: "主版本格式错误", query: "tlsMinVersion=2.0", wantErr: "tlsMinVersion.*格式"},
		{name: "版本号溢出", query: "tlsMinVersion=1.65535", wantErr: "tlsMinVersion.*超出"},
		{name: "拒绝反向范围", query: "tlsMinVersion=1.3&tlsMaxVersion=1.2", wantErr: "tlsMinVersion.*不能大于.*tlsMaxVersion"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extra := MustValue(t, func() (url.Values, error) { return url.ParseQuery(tc.query) })
			config := &Config{Endpoint: strfmt.Endpoint{Scheme: "ftps", Hostname: "127.0.0.1", Extra: extra}}
			// 在实际连接前取消，只验证 Conn 对端点参数的解析和初始化。
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := config.Conn(ctx)
			if tc.wantErr != "" {
				Then(t, "非法配置在连接前返回错误",
					ExpectDo(func() error { return err }, ErrorMatch(regexp.MustCompile(tc.wantErr))),
					Expect(config.p, Equal((*Pool)(nil))),
				)
				return
			}
			Then(t, "初始化 TLS 配置", Expect(config.p != nil, Equal(true)))
			if config.p == nil {
				return
			}
			Then(t, "TLS 版本范围符合端点参数",
				Expect(config.p.TLSConfig.MinVersion, Equal(tc.min)),
				Expect(config.p.TLSConfig.MaxVersion, Equal(tc.max)),
			)
		})
	}
}

func TestConfigTLSVersionHandshake(t *testing.T) {
	certificateServer := httptest.NewTLSServer(nil)
	serverTLS := certificateServer.TLS.Clone()
	certificateServer.Close()

	for _, tc := range []struct {
		name       string
		query      string
		server     uint16
		wantReject bool
	}{
		{name: "默认允许 TLS 1.3", server: tls.VersionTLS13},
		{name: "固定 TLS 1.2 可以连接 TLS 1.2", query: "tlsMinVersion=1.2&tlsMaxVersion=1.2", server: tls.VersionTLS12},
		{name: "上限 TLS 1.2 拒绝 TLS 1.3", query: "tlsMaxVersion=1.2", server: tls.VersionTLS13, wantReject: true},
		{name: "下限 TLS 1.3 拒绝 TLS 1.2", query: "tlsMinVersion=1.3", server: tls.VersionTLS12, wantReject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverConfig := serverTLS.Clone()
			serverConfig.MinVersion = tc.server
			serverConfig.MaxVersion = tc.server
			listener := MustValue(t, func() (net.Listener, error) {
				return tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			})
			t.Cleanup(func() { _ = listener.Close() })
			type result struct {
				version uint16
				err     error
			}
			completed := make(chan result, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					completed <- result{err: err}
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				tlsConn := conn.(*tls.Conn)
				err = tlsConn.Handshake()
				completed <- result{version: tlsConn.ConnectionState().Version, err: err}
			}()

			endpoint := MustValue(t, func() (*strfmt.Endpoint, error) {
				return strfmt.ParseEndpoint("ftps://" + listener.Addr().String() + "?insecureSkipVerify=true&" + tc.query)
			})
			config := &Config{Endpoint: *endpoint}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// 服务端在 TLS 握手后关闭；不模拟后续 FTP 登录协议。
			_, _ = config.Conn(ctx)
			select {
			case got := <-completed:
				if tc.wantReject {
					Then(t, "服务端拒绝不兼容的版本",
						ExpectDo(func() error { return got.err }, ErrorMatch(regexp.MustCompile("unsupported versions"))),
					)
				} else {
					Then(t, "使用预期 TLS 版本完成握手",
						Expect(got.err, Equal[error](nil)),
						Expect(got.version, Equal(tc.server)),
					)
				}
			case <-ctx.Done():
				t.Fatal("等待 TLS 握手结果超时")
			}
		})
	}
}
