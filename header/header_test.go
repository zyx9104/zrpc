package header

import (
	"bufio"
	"bytes"
	"log"
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestRequestHeader(t *testing.T) {

	cases := []struct {
		Name string
		Req  *RequestHeader
		Res  *RequestHeader
	}{
		{
			Name: "test-1",
			Req: &RequestHeader{
				MethodLen:     8,
				CompressType:  1,
				RequestLen:    10,
				Seq:           1,
				ServiceMethod: "Test.Add",
			},
			Res: &RequestHeader{},
		},
	}

	for _, c := range cases {
		buf := &bytes.Buffer{}

		wt := bufio.NewWriter(buf)
		rd := bufio.NewReader(buf)
		if err := c.Req.Write(wt); err != nil {
			log.Fatal(err)
		}

		if err := c.Res.Read(rd); err != nil {
			log.Fatal(err)
		}
		assert.Equal(t, c.Req, c.Res)
	}

}

func TestResponseHeader(t *testing.T) {

	cases := []struct {
		Name string
		Req  *ResponseHeader
		Res  *ResponseHeader
	}{
		{
			Name: "test-1",
			Req: &ResponseHeader{
				ErrorLen:     8,
				CompressType: 1,
				ResponseLen:  10,
				Seq:          1,
				Error:        "no error",
			},
			Res: &ResponseHeader{},
		},
	}

	for _, c := range cases {
		buf := &bytes.Buffer{}

		wt := bufio.NewWriter(buf)
		rd := bufio.NewReader(buf)
		if err := c.Req.Write(wt); err != nil {
			log.Fatal(err)
		}

		if err := c.Res.Read(rd); err != nil {
			log.Fatal(err)
		}
		assert.Equal(t, c.Req, c.Res)
	}

}
