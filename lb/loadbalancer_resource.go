// Copyright (c) 2017, Oracle and/or its affiliates. All rights reserved.

package lb

import (
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/oracle/terraform-provider-baremetal/client"
)

func LoadBalancerResource() *schema.Resource {
	return &schema.Resource{
		Create: createLoadBalancer,
		Read:   readLoadBalancer,
		Update: updateLoadBalancer,
		Delete: deleteLoadBalancer,
		Schema: map[string]*schema.Schema{
			// Required {
			"compartment_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"shape": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_ids": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"display_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			// }
			// Computed {
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"time_created": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createLoadBalancer(d *schema.ResourceData, m interface{}) (e error) {
	sync := LoadBalancerResourceCrud{
		D:      d,
		Client: m.(client.BareMetalClient),
	}
	return sync.Create()
}

func readLoadBalancer(d *schema.ResourceData, m interface{}) (e error) {
	sync := &LoadBalancerResourceCrud{
		D:      d,
		Client: m.(client.BareMetalClient),
	}
	return sync.Read()
}

func updateLoadBalancer(d *schema.ResourceData, m interface{}) (e error) {
	sync := LoadBalancerResourceCrud{
		D:      d,
		Client: m.(client.BareMetalClient),
	}
	return sync.Update()
}

func deleteLoadBalancer(d *schema.ResourceData, m interface{}) (e error) {
	sync := &LoadBalancerResourceCrud{
		D:      d,
		Client: m.(client.BareMetalClient),
	}
	return sync.Delete()
}
