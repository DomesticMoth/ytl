// Ytl by DomesticMoth
//
// To the extent possible under law, the person who associated CC0 with
// ytl has waived all copyright and related or neighboring rights
// to ytl.
//
// You should have received a copy of the CC0 legalcode along with this
// work.  If not, see <http://creativecommons.org/publicdomain/zero/1.0/>.
package ytl

import (
	"net"
	"net/url"
	"time"
	"context"
	"encoding/hex"
	"crypto/ed25519"
	"github.com/DomesticMoth/ytl/ytl/transports"
	"github.com/DomesticMoth/ytl/ytl/static"
)

func materialise(key ed25519.PrivateKey) ed25519.PrivateKey {
	if key != nil {
		return key
	}
	_, spriv, err := ed25519.GenerateKey(nil)
	if err != nil { panic(err) }
	return spriv
}

type YggConn struct{
	innerConn net.Conn
	transport_key ed25519.PublicKey
	allowList *static.AllowList
	secureTranport bool
	ctx context.Context
}

func connToYggConn(ctx context.Context, conn net.Conn, transport_key ed25519.PublicKey, allow *static.AllowList, secureTranport bool) *YggConn {
	if conn == nil {return nil}
	return &YggConn{conn, transport_key, allow, secureTranport, ctx}
}

func (y * YggConn) Close() error {
	return y.innerConn.Close()
}

func (y * YggConn) Read(b []byte) (n int, err error) {
	return y.innerConn.Read(b)
}

//Write(b []byte) (n int, err error)
//LocalAddr() Addr
//RemoteAddr() Addr
//SetDeadline(t time.Time) error
//SetReadDeadline(t time.Time) error
//SetWriteDeadline(t time.Time) error

/*type YggListener struct {
	inner_listener net.Listener
}*/

type ConnManager struct{
	transports map[string]static.Transport
	key ed25519.PrivateKey
	proxyManager ProxyManager
	allowList *static.AllowList
	ctx context.Context
}

func NewConnManager(ctx context.Context, key ed25519.PrivateKey, proxy *ProxyManager, allowList *static.AllowList) *ConnManager{
	transports_list := make(map[string]static.Transport)
	for _, transport := range transports.TransportsList{
		transports_list[transport.GetScheme()] = transport
	}
	if proxy == nil {
		p := NewProxyManager(nil, nil)
		proxy = &p
	}
	return &ConnManager{transports_list, key, *proxy, allowList, ctx}
}

func (c * ConnManager) innerConnect(ctx context.Context, uri url.URL) (*YggConn, error) {
	var allowList *static.AllowList = nil
	if c.allowList != nil {
		allow := make(static.AllowList, len(*c.allowList))
		copy(allow, *c.allowList)
		allowList = &allow
	}
	if pubkeys, ok := uri.Query()["key"]; ok && len(pubkeys) > 0 {
		var allow static.AllowList
		if allowList == nil {
			allow = make(static.AllowList, 0)
		}else{
			allow = *allowList
		}
		for _, pubkey := range pubkeys {
			if key, err := hex.DecodeString(pubkey); err == nil {
				allow = append(allow, key)
			}
		}
		allowList = &allow
	}
	if transport, ok := c.transports[uri.Scheme]; ok {
		conn, transport_key, err := transport.Connect(ctx, uri, c.proxyManager.Get(uri), materialise(c.key))
		if allowList != nil {
			if allowList.IsAllow(transport_key) {
				conn.Close()
					return nil, static.IvalidPeerPublicKey{
						Text: "Key received from the peer is not in the allow list",
					}
			}
		}
		return connToYggConn(c.ctx, conn, transport_key, allowList, transport.IsSecure()), err
	}
	return nil, static.UnknownSchemeError{Scheme: uri.Scheme}
}

func (c * ConnManager) Connect(uri url.URL) (*YggConn, error) {
	return c.innerConnect(c.ctx, uri)
}

func (c * ConnManager) ConnectStr(uri string) (*YggConn, error) {
	u, err := url.Parse(uri)
	if err != nil { return nil, err }
	return c.innerConnect(c.ctx, *u)
}

func (c * ConnManager) ConnectCtx(ctx context.Context, uri url.URL) (*YggConn, error) {
	return c.innerConnect(ctx, uri)
}

func (c * ConnManager) ConnectCtxStr(ctx context.Context, uri string) (*YggConn, error) {
	u, err := url.Parse(uri)
	if err != nil { return nil, err }
	return c.innerConnect(ctx, *u)
}

func (c * ConnManager) ConnectTimeout(uri url.URL, timeout time.Duration) (*YggConn, error) {
	type Result struct{
		Conn *YggConn
		Error error
	}
    result := make(chan Result, 1)
    ctx, cancel := context.WithTimeout(c.ctx, timeout)
    go func() {
    	conn, err := c.innerConnect(ctx, uri)
        result <- Result{conn, err}
    }()
    select {
    	case <-time.After(timeout):
    		cancel()
    		go func() {
    			result := <-result
    			if result.Error == nil && result.Conn != nil{
    				result.Conn.Close()
    			}
    		}()
        	return nil, static.ConnTimeoutError{}
    	case result := <-result:
    		cancel()
        	return result.Conn, result.Error
    }
}

func (c * ConnManager) ConnectTimeoutStr(uri string, timeout time.Duration) (*YggConn, error) {
	u, err := url.Parse(uri)
	if err != nil { return nil, err }
	return c.ConnectTimeout(*u, timeout)
}

//func (c * ConnManager) Listen(uri string) (net.Listener, error) {
//	
//}
