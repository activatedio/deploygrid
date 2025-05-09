package service

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type gridService struct{}

func (g *gridService) Get(ctx context.Context) (*deploygrid.Grid, error) {
	return &deploygrid.Grid{
		Environments: []deploygrid.Environment{
			{
				Name: "dev",
			},
			{
				Name: "qa",
			},
			{
				Name: "stage",
			},
		},
		Components: []deploygrid.Component{
			{
				Name:          "Apps",
				ComponentType: "Group",
				Children: []deploygrid.Component{
					{
						Name:          "Corp",
						ComponentType: "Group",
						Children: []deploygrid.Component{
							{
								Name:          "app-1",
								ComponentType: "HelmChart",
								Deployments: map[string]deploygrid.Deployment{
									"dev": {
										Version: "1.0.0",
									},
									"qa": {
										Version: "1.0.1",
									},
									"stage": {
										Version: "1.0.2",
									},
								},
								Children: []deploygrid.Component{
									{
										Name:          "app-1-main",
										ComponentType: "Container",
										Deployments: map[string]deploygrid.Deployment{
											"dev": {
												Version: "0.0.1",
											},
											"qa": {
												Version: "0.0.2",
											},
											"stage": {
												Version: "0.0.3",
											},
										},
									},
									{
										Name:          "app-1-workhorse",
										ComponentType: "Container",
										Deployments: map[string]deploygrid.Deployment{
											"dev": {
												Version: "0.0.1",
											},
											"qa": {
												Version: "0.0.2",
											},
											"stage": {
												Version: "0.0.3",
											},
										},
									},
								},
							},
						},
					},
					{
						Name:          "Customer",
						ComponentType: "Group",
						Children: []deploygrid.Component{
							{
								Name:          "cust-app-1",
								ComponentType: "HelmChart",
								Deployments: map[string]deploygrid.Deployment{
									"dev": {
										Version: "1.0.0",
									},
									"qa": {
										Version: "1.0.1",
									},
									"stage": {
										Version: "1.0.2",
									},
								},
								Children: []deploygrid.Component{
									{
										Name:          "cust-app-1-main",
										ComponentType: "Container",
										Deployments: map[string]deploygrid.Deployment{
											"dev": {
												Version: "0.0.1",
											},
											"qa": {
												Version: "0.0.2",
											},
											"stage": {
												Version: "0.0.3",
											},
										},
									},
									{
										Name:          "cust-app-1-workhorse",
										ComponentType: "Container",
										Deployments: map[string]deploygrid.Deployment{
											"dev": {
												Version: "0.0.1",
											},
											"qa": {
												Version: "0.0.2",
											},
											"stage": {
												Version: "0.0.3",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Name:          "Foundation",
				ComponentType: "Group",
				Children: []deploygrid.Component{
					{
						Name:          "sealed-secrets",
						ComponentType: "HelmChart",
						Deployments: map[string]deploygrid.Deployment{
							"dev": {
								Version: "1.0.0",
							},
							"qa": {
								Version: "1.0.1",
							},
							"stage": {
								Version: "1.0.2",
							},
						},
						Children: []deploygrid.Component{
							{
								Name:          "sealed-secrets-main",
								ComponentType: "Container",
								Deployments: map[string]deploygrid.Deployment{
									"dev": {
										Version: "0.0.1",
									},
									"qa": {
										Version: "0.0.2",
									},
									"stage": {
										Version: "0.0.3",
									},
								},
							},
							{
								Name:          "sealed-secrets-sidecar",
								ComponentType: "Container",
								Deployments: map[string]deploygrid.Deployment{
									"dev": {
										Version: "0.0.1",
									},
									"qa": {
										Version: "0.0.2",
									},
									"stage": {
										Version: "0.0.3",
									},
								},
							},
						},
					},
					{
						Name:          "external-dns",
						ComponentType: "HelmChart",
						Deployments: map[string]deploygrid.Deployment{
							"dev": {
								Version: "1.0.0",
							},
							"qa": {
								Version: "1.0.1",
							},
							"stage": {
								Version: "1.0.2",
							},
						},
					},
					{
						Name:          "alb-controller",
						ComponentType: "HelmChart",
						Deployments: map[string]deploygrid.Deployment{
							"dev": {
								Version: "1.0.0",
							},
							"qa": {
								Version: "1.0.1",
							},
							"stage": {
								Version: "1.0.2",
							},
						},
					},
				},
			},
		},
	}, nil
}

func NewGridService() GridService {
	return &gridService{}
}
