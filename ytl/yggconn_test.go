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
	"io"
	"net"
	"bytes"
	"testing"
	"crypto/ed25519"
	"github.com/DomesticMoth/ytl/ytl/static"
)

func MokeKey() []byte {
	buf := MokeConnContent()
	return buf[len(buf)-ed25519.PublicKeySize:]
}

func MokeConnContent() []byte {
	return []byte{
		109, 101, 116, 97, // 'm' 'e' 't' 'a'
		0, 4, // Version
		// PublicKey
		194, 220, 146,  21, 237, 163, 168,  31,
		216,  91, 173,   6,  46, 225, 161, 231,
		146, 238,  83, 130, 131,  95, 151, 141,
		143,  73, 142,  61,  27, 142, 160, 212,
	}
}

func TestYggConnCorrectReading(t *testing.T){
	for i := 0; i < 2; i++ {
		strictMode := false
		if i > 0 { strictMode = true }
		for j := 0; j < 2; j++ {
			secure := false
			if j > 0 { secure = true }
			data := MokeConnContent()
			a, b := net.Pipe()
			dm := NewDeduplicationManager(strictMode)
			yggcon := ConnToYggConn(
				a,
				MokeKey(),
				nil,
				secure,
				dm,
			)
			defer yggcon.Close()
			go func() {
				b.Write(data)
				b.Close()
			}()
			buf := make([]byte, len(data))
			n, err := io.ReadFull(yggcon, buf)
			if err != nil {
				t.Errorf("Error while reading from yggcon '%s'", err)
				t.Errorf("Readed only %d bytes from %d", n, len(data))
			}
			buf = buf[:n]
			if bytes.Compare(data, buf) != 0 {
				t.Errorf("Readed data is not eq to writed data")
			}
			target_version := static.PROTO_VERSION()
			version, err := yggcon.GetVer()
			if err != nil {
				t.Errorf("Error while reading verdion %s", err)
			}else{
				if version.Major != target_version.Major || version.Minor != target_version.Minor {
					t.Errorf("Invalid version")
				}
			}
			key, err := yggcon.GetPublicKey()
			if err != nil {
				t.Errorf("Error while reading public key %s", err)
			}else{
				if bytes.Compare(key, MokeKey()) != 0 {
					t.Errorf("Invalid key")
				}
			}
		}
	}
}

func yggConnTestCollision(
		t *testing.T, 
		n1, n2 bool, // secure params for connections
		n int, // The number of the connection to be CLOSED
	) {
	dm := NewDeduplicationManager(true)
	data := MokeConnContent()
	a, c := net.Pipe()
	b, d := net.Pipe()
	yggcon1 := ConnToYggConn(
		a,
		MokeKey(),
		nil,
		n1,
		dm,
	)
	yggcon2 := ConnToYggConn(
		b,
		MokeKey(),
		nil,
		n2,
		dm,
	)
	defer yggcon1.Close()
	defer yggcon2.Close()
	defer d.Close()
	defer c.Close()
	go func() {
		c.Write(data)
	}()
	go func() {
		d.Write(data)
	}()
	buf := make([]byte, 1)
	var err1, err2 error
	if n == 1 {
		_, err1 = io.ReadFull(yggcon2, buf)
		_, err2 = io.ReadFull(yggcon1, buf)
	}else{
		_, err1 = io.ReadFull(yggcon1, buf)
		_, err2 = io.ReadFull(yggcon2, buf)
	}
	t.Errorf("e1 %s; e2 %s", err1, err2)
	return
	if err1 != nil {
		t.Errorf("Wrong conn was closed %s", err1)
		return
	}
	if err2 == nil {
		t.Errorf("Connection was not closed")
		return
	}
	switch err2.(type) {
		case static.ConnClosedByDeduplicatorError:
		    // Ok
		default:
		    t.Errorf("Connection was not closed by deduplicator %s", err2)
		    return
	}
}

func TestYggConnCollisionII(t *testing.T){
	yggConnTestCollision(t, false, false, 2)
}

/*func TestYggConnTestCollisionIS(t *testing.T){
	yggConnTestCollision(t, false, true, 1)
}

func TestYggConnTestCollisionSI(t *testing.T){
	yggConnTestCollision(t, true, false, 2)
}

func TestYggConnTestCollisionSS(t *testing.T){
	yggConnTestCollision(t, true, false, 2)
}*/

/*
func YggConnTestCollisionIS2(t *testing.T){
	n1 := false
	n2 := true
	n := 1
	yggConnTestCollision(t, n1, n2, n)
}*/

/*func TestYggConnDedplication(t *testing.T){
	dm := NewDeduplicationManager(true)
	data := MokeConnContent()
	a, c := net.Pipe()
	b, d := net.Pipe()
	yggcon1 := ConnToYggConn(
		a,
		MokeKey(),
		nil,
		false,
		dm,
	)
	yggcon2 := ConnToYggConn(
		b,
		MokeKey(),
		nil,
		true,
		dm,
	)
	defer yggcon1.Close()
	defer yggcon2.Close()
	defer d.Close()
	defer c.Close()
	go func() {
		c.Write(data)
	}()
	go func() {
		d.Write(data)
	}()
	buf := make([]byte, 1)
	io.ReadFull(yggcon2, buf)
	_, err := io.ReadFull(yggcon1, buf)
	if err == nil {
		t.Errorf("Connection is not closed")
		return
	}
	switch err.(type) {
		case static.ConnClosedByDeduplicatorError:
		    // Ok
		default:
		    t.Errorf("Connection is not closed by deduplicator %s", err)
		    return
	}
}*/

/*func TestYggConnDedplication2(t *testing.T){
	n1 := false
	n2 := true
	n := 1
	yggConnTestCollision(t, n1, n2, n)
	/*dm := NewDeduplicationManager(true)
	data := MokeConnContent()
	a, c := net.Pipe()
	b, d := net.Pipe()
	yggcon1 := ConnToYggConn(
		a,
		MokeKey(),
		nil,
		n1,
		dm,
	)
	yggcon2 := ConnToYggConn(
		b,
		MokeKey(),
		nil,
		n2,
		dm,
	)
	defer yggcon1.Close()
	defer yggcon2.Close()
	defer d.Close()
	defer c.Close()
	go func() {
		c.Write(data)
	}()
	go func() {
		d.Write(data)
	}()
	buf := make([]byte, 1)
	var err1, err2 error
	if n == 1 {
		_, err1 = io.ReadFull(yggcon2, buf)
		_, err2 = io.ReadFull(yggcon1, buf)
	}else{
		_, err1 = io.ReadFull(yggcon1, buf)
		_, err2 = io.ReadFull(yggcon2, buf)
	}
	if err1 != nil {
		t.Errorf("Wrong conn was closed %s", err1)
		return
	}
	if err2 == nil {
		t.Errorf("Connection was not closed")
		return
	}
	switch err2.(type) {
		case static.ConnClosedByDeduplicatorError:
		    // Ok
		default:
		    t.Errorf("Connection was not closed by deduplicator %s", err2)
	}
}*/