package propsigner

import (
	"errors"
	"io"

	"github.com/multiformats/go-varint"
	"google.golang.org/protobuf/proto"
)

type byteReader struct {
	r io.Reader
}

func (d *byteReader) ReadByte() (byte, error) {
	var buf [1]byte
	_, err := d.r.Read(buf[:])
	return buf[0], err
}

func readMsg(r io.Reader, maxSize int, msg proto.Message) error {
	mlen, err := varint.ReadUvarint(&byteReader{r: r})
	if err != nil {
		return err
	}

	if uint64(maxSize) < mlen {
		return errors.New("message too large")
	}

	buf := make([]byte, maxSize)
	buf = buf[:mlen]
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}

	return proto.Unmarshal(buf, msg)
}

func writeMsg(w io.Writer, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint64(len(data))
	uw := make([]byte, varint.MaxLenUvarint63)
	n := varint.PutUvarint(uw, length)
	_, err = w.Write(uw[:n])
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
