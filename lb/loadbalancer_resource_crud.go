// Copyright (c) 2017, Oracle and/or its affiliates. All rights reserved.

package lb

import (
	"fmt"
	"log"
	"strings"

	"github.com/MustWin/baremetal-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/oracle/terraform-provider-baremetal/client"
	"github.com/oracle/terraform-provider-baremetal/crud"
)

// LoadBalancerResourceCrud wraps a baremetal.LoadBalancer to support crud
type LoadBalancerResourceCrud struct {
	D           *schema.ResourceData
	Client      client.BareMetalClient
	WorkRequest *baremetal.WorkRequest
	Resource    *baremetal.LoadBalancer
}

// Create makes a request to create a new load balancer from the resourceData
// It should leave the work request set up
func (s *LoadBalancerResourceCrud) Create() error {
	rawNets := s.D.Get("subnet_ids").([]interface{})
	sns := make([]string, len(rawNets))
	for i, v := range rawNets {
		sns[i] = v.(string)
	}

	opts := &baremetal.CreateOptions{}
	opts.DisplayName = s.D.Get("display_name").(string)

	workReqID, err := s.Client.CreateLoadBalancer(
		nil,
		nil,
		s.D.Get("compartment_id").(string),
		nil,
		s.D.Get("shape").(string),
		sns,
		opts)
	if err != nil {
		return err
	}

	wr, err := s.Client.GetWorkRequest(workReqID, nil)
	if err != nil {
		return err
	}
	s.WorkRequest = wr

	if err := s.D.Set("state", s.State()); err != nil {
		return err
	}

	// ID is required for state refresh
	s.D.SetId(s.ID())
	if err := s.WaitForCreatedState(); err != nil {
		return err
	}

	s.D.SetId(s.ID())
	s.SetData()

	return nil
}

func (s *LoadBalancerResourceCrud) Read() error {
	if err := s.Get(); err != nil {
		crud.FilterMissingResourceError(s, &err)
		return err
	}
	s.SetData()

	return nil
}

// Update makes a request to update the load balancer
func (s *LoadBalancerResourceCrud) Update() error {
	s.D.Partial(true)

	opts := &baremetal.UpdateOptions{}
	if displayName, ok := s.D.GetOk("display_name"); ok {
		opts.DisplayName = displayName.(string)
	}

	workReqID, err := s.Client.UpdateLoadBalancer(s.D.Id(), opts)
	if err != nil {
		return err
	}

	wr, err := s.Client.GetWorkRequest(workReqID, nil)
	if err != nil {
		return err
	}
	s.WorkRequest = wr

	s.D.Partial(false)
	s.SetData()

	return err
}

// Delete makes a request to delete the load balancer
func (s *LoadBalancerResourceCrud) Delete() error {
	workReqID, err := s.Client.DeleteLoadBalancer(s.D.Id(), nil)
	if err != nil {
		return err
	}

	wr, err := s.Client.GetWorkRequest(workReqID, nil)
	if err != nil {
		return err
	}
	s.WorkRequest = wr

	if err = s.WaitForDeletedState(); err != nil {
		crud.FilterMissingResourceError(s, &err)
	} else {
		s.VoidState()
	}

	return err
}

// ID delegates to the load balancer ID, falling back to the work request ID
func (s *LoadBalancerResourceCrud) ID() string {
	log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID()")
	log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: Resource: %#v", s.Resource)
	if s.Resource != nil && s.Resource.ID != "" {
		log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: Resource.ID: %#v", s.Resource.ID)
		return s.Resource.ID
	}
	log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: WorkRequest: %#v", s.WorkRequest)
	if s.WorkRequest != nil {
		log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: WorkRequest.State: %s", s.WorkRequest.State)
		if s.WorkRequest.State == baremetal.WorkRequestSucceeded {
			log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: WorkRequest.LoadBalancerID: %#v", s.WorkRequest.LoadBalancerID)
			return s.WorkRequest.LoadBalancerID
		} else {
			log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: WorkRequest.ID: %s", s.WorkRequest.ID)
			return s.WorkRequest.ID
		}
	}
	log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.ID: Resource & WorkRequest are nil, returning \"\"")
	return ""
}

// Get makes a request to get the load balancer, populating s.Resource.
// May modify s.Id & s.D.Get("state"). Does not populate anything else in s.D.
func (s *LoadBalancerResourceCrud) Get() (e error) {
	// key: {workRequestID} || {loadBalancerID}
	id := s.D.Id()
	log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: ID: %#v", id)
	if id == "" {
		panic(fmt.Sprintf("LoadBalancer had empty ID: %#v Resource: %#V", s, s.Resource))
	}
	wr := s.WorkRequest
	log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: WorkRequest: %#v", wr)
	state := s.D.Get("state").(string)
	log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: State: %#v", state)

	// NOTE: if the id is for a work request, refresh its state and loadBalancerID. then refresh the load balancer
	if strings.HasPrefix(id, "ocid1.loadbalancerworkrequest.") {
		log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: ID is for WorkRequest, refreshing")
		s.WorkRequest, e = s.Client.GetWorkRequest(id, nil)
		log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: WorkRequest: %#v", s.WorkRequest)
		s.D.Set("state", s.WorkRequest.State)
		if s.WorkRequest.State == baremetal.WorkRequestSucceeded {
			id = s.WorkRequest.LoadBalancerID
			if id == "" {
				panic(fmt.Sprintf("WorkRequest had empty LoadBalancerID: %#v", s.WorkRequest))
			}
			s.D.SetId(id)
			// unset work request on success
			s.WorkRequest = nil
		} else {
			// We do not have a LoadBalancerID, so we short-circuit out
			return

		}
	}

	if !strings.HasPrefix(id, "ocid1.loadbalancer.") {
		panic(fmt.Sprintf("Cannot request loadbalancer with this ID, expected it to begin with \"ocid1.loadbalancer.\", but was: %#v", id))
	}
	log.Printf("[DEBUG] lb.LoadBalancerBackendResource.Get: ID: %#v", id)
	if id == "" {
		panic(fmt.Sprintf("LoadBalancer had empty ID: %#v Resource: %#V", s, s.Resource))
	}
	s.Resource, e = s.Client.GetLoadBalancer(id, nil)

	return
}

// SetData populates the resourceData from the model
func (s *LoadBalancerResourceCrud) SetData() {
	// The first time this is called, we haven't actually fetched the resource yet, we just got a work request
	if s.Resource != nil && s.Resource.ID != "" {
		s.D.SetId(s.Resource.ID)
		s.D.Set("compartment_id", s.Resource.CompartmentID)
		s.D.Set("display_name", s.Resource.DisplayName)
		s.D.Set("shape", s.Resource.Shape)
		s.D.Set("subnet_ids", s.Resource.SubnetIDs)
		// Computed
		s.D.Set("id", s.Resource.ID)
		s.D.Set("state", s.Resource.State)
		s.D.Set("time_created", s.Resource.TimeCreated.String())
		ip_addresses := make([]string, len(s.Resource.IPAddresses))
		for i, ad := range s.Resource.IPAddresses {
			ip_addresses[i] = ad.IPAddress
		}
		s.D.Set("ip_addresses", ip_addresses)
	}
}

func (s *LoadBalancerResourceCrud) WaitForCreatedState() error {
	conf := &resource.StateChangeConf{
		Pending: []string{
			baremetal.ResourceWaitingForWorkRequest,
			baremetal.ResourceCreating,
		},
		Target: []string{
			baremetal.ResourceActive,
		},
		Timeout: s.D.Timeout(schema.TimeoutCreate),
		Refresh: func() (result interface{}, state string, err error) {
			if err = s.Get(); err != nil {
				return nil, "", err
			}
			return s, s.State(), nil
		},
	}

	if _, err := conf.WaitForState(); err != nil {
		crud.FilterMissingResourceError(s, &err)
		return err
	}

	return nil
}

func (s *LoadBalancerResourceCrud) WaitForDeletedState() error {
	conf := &resource.StateChangeConf{
		Pending: []string{
			baremetal.ResourceWaitingForWorkRequest,
			baremetal.ResourceDeleting,
		},
		Target: []string{
			baremetal.ResourceDeleted,
		},
		Timeout: s.D.Timeout(schema.TimeoutDelete),
		Refresh: func() (result interface{}, state string, err error) {
			if err = s.Get(); err != nil {
				return nil, "", err
			}
			return s, s.State(), nil
		},
	}

	if _, err := conf.WaitForState(); err != nil {
		crud.FilterMissingResourceError(s, &err)
		return err
	}

	return nil
}

// State returns the current state of the load balancer resource (if present), of the work request (if present), or an empty string
func (s *LoadBalancerResourceCrud) State() string {
	if s.Resource != nil {
		return s.Resource.State
	}
	if s.WorkRequest != nil {
		return s.WorkRequest.State
	}
	log.Printf("[DEBUG] lb.LoadBalancerResourceCrud.State: Resource & WorkRequest are nil, returning \"\"")
	return ""
}

func (s *LoadBalancerResourceCrud) VoidState() {
	s.D.SetId("")
}
