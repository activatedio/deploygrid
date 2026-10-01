package main

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type printer struct {
	documentVisitor DocumentVisitor
	nodeVisitor     NodeVisitor
}

func walk(v NodeVisitor, n *yaml.Node, ctx any) (any, error) {

	ctx, next, err := v.Visit(n, ctx)

	if err != nil {
		return nil, err
	}

	if next == nil {
		next = v
	}

	var childCtx any

	for _, nn := range n.Content {
		childCtx, err = walk(next, nn, childCtx)
		if err != nil {
			return nil, err
		}
	}
	return ctx, nil

}

func (p *printer) run(in io.Reader) error {

	bs, err := p.modify(in)

	if err != nil {
		return err
	}

	fmt.Println(string(bs))

	return nil
}

func (p *printer) modify(in io.Reader) ([]byte, error) {

	out := map[string]any{}

	err := yaml.NewDecoder(in).Decode(&out)

	if err != nil {
		return nil, err
	}

	delete(out, "status")
	delete(out["metadata"].(map[string]any), "creationTimestamp")

	err = p.documentVisitor.Visit(out)

	if err != nil {
		return nil, err
	}

	var bs []byte
	bs, err = yaml.Marshal(out)

	if err != nil {
		return nil, err
	}

	n := &yaml.Node{}

	err = yaml.Unmarshal(bs, n)

	if err != nil {
		return nil, err
	}

	_, err = walk(p.nodeVisitor, n, nil)

	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	buf.WriteString("---\n")

	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	err = enc.Encode(n)

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
