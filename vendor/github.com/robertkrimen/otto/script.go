package otto

import (
	"bytes"
	"encoding/gob"
	"errors"
)

var ErrVersion = errors.New("version mismatch")

var scriptVersion = "2014-04-13/1"

type Script struct {
	version  string
	program  *_nodeProgram
	filename string
	src      string
}

func (self *Otto) Compile(filename string, src interface{}) (*Script, error) {
	return self.CompileWithSourceMap(filename, src, nil)
}

func (self *Otto) CompileWithSourceMap(filename string, src, sm interface{}) (*Script, error) {
	program, err := self.runtime.parse(filename, src, sm)
	if err != nil {
		return nil, err
	}

	cmpl_program := cmpl_parse(program)

	script := &Script{
		version:  scriptVersion,
		program:  cmpl_program,
		filename: filename,
		src:      program.File.Source(),
	}

	return script, nil
}

func (self *Script) String() string {
	return "// " + self.filename + "\n" + self.src
}

func (self *Script) marshalBinary() ([]byte, error) {
	var bfr bytes.Buffer
	encoder := gob.NewEncoder(&bfr)
	err := encoder.Encode(self.version)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(self.program)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(self.filename)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(self.src)
	if err != nil {
		return nil, err
	}
	return bfr.Bytes(), nil
}

func (self *Script) unmarshalBinary(data []byte) error {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&self.version)
	if err != nil {
		goto error
	}
	if self.version != scriptVersion {
		err = ErrVersion
		goto error
	}
	err = decoder.Decode(&self.program)
	if err != nil {
		goto error
	}
	err = decoder.Decode(&self.filename)
	if err != nil {
		goto error
	}
	err = decoder.Decode(&self.src)
	if err != nil {
		goto error
	}
	return nil
error:
	self.version = ""
	self.program = nil
	self.filename = ""
	self.src = ""
	return err
}
