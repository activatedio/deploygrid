package main

import (
	"gopkg.in/yaml.v3"
)

type NodeVisitor interface {
	// Returns nodeVisitor for children
	Visit(node *yaml.Node, ctx any) (any, NodeVisitor, error)
}

type DocumentVisitor interface {
	Visit(doc map[string]any) error
}

type NodeVisitorFunc func(node *yaml.Node, ctx any) (any, NodeVisitor, error)

type nodeVisitorFuncShunt struct {
	nodeVisitorFunc NodeVisitorFunc
}

func (v *nodeVisitorFuncShunt) Visit(node *yaml.Node, ctx any) (any, NodeVisitor, error) {
	return v.nodeVisitorFunc(node, ctx)
}

func NewNodeVisitorFunc(fn NodeVisitorFunc) NodeVisitor {
	return &nodeVisitorFuncShunt{
		nodeVisitorFunc: fn,
	}
}
