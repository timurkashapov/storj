// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package kademlia

import (
	"context"

	"storj.io/storj/pkg/identity"
	"storj.io/storj/pkg/pb"
	"storj.io/storj/pkg/storj"
)

// Inspector is a gRPC service for inspecting kademlia internals
type Inspector struct {
	dht      *Kademlia
	identity *identity.FullIdentity
}

// NewInspector creates an Inspector
func NewInspector(kad *Kademlia, identity *identity.FullIdentity) *Inspector {
	return &Inspector{
		dht:      kad,
		identity: identity,
	}
}

// CountNodes returns the number of nodes in the routing table
func (srv *Inspector) CountNodes(ctx context.Context, req *pb.CountNodesRequest) (*pb.CountNodesResponse, error) {
	// TODO: this is definitely the wrong way to get this
	kadNodes, err := srv.dht.FindNear(ctx, srv.identity.ID, 100000)
	if err != nil {
		return nil, err
	}

	return &pb.CountNodesResponse{
		Count: int64(len(kadNodes)),
	}, nil
}

// GetBuckets returns all kademlia buckets for current kademlia instance
func (srv *Inspector) GetBuckets(ctx context.Context, req *pb.GetBucketsRequest) (*pb.GetBucketsResponse, error) {
	b, err := srv.dht.GetBucketIds()
	if err != nil {
		return nil, err
	}
	// TODO(bryanchriswhite): should use bucketID type
	nodeIDs, err := storj.NodeIDsFromBytes(b.ByteSlices())
	if err != nil {
		return nil, err
	}
	return &pb.GetBucketsResponse{
		Total: int64(len(b)),
		// TODO(bryanchriswhite): should use bucketID type
		Ids: nodeIDs,
	}, nil
}

// FindNear sends back limit of near nodes
func (srv *Inspector) FindNear(ctx context.Context, req *pb.FindNearRequest) (*pb.FindNearResponse, error) {
	start := req.Start
	limit := req.Limit
	nodes, err := srv.dht.FindNear(ctx, start, int(limit))
	if err != nil {
		return &pb.FindNearResponse{}, err
	}
	return &pb.FindNearResponse{
		Nodes: nodes,
	}, nil
}

// PingNode sends a PING RPC to the provided node ID in the Kad network.
func (srv *Inspector) PingNode(ctx context.Context, req *pb.PingNodeRequest) (*pb.PingNodeResponse, error) {
	self := srv.dht.Local()

	_, err := srv.dht.Ping(ctx, pb.Node{
		Id:   req.Id,
		Type: self.Type,
		Address: &pb.NodeAddress{
			Address: req.Address,
		},
	})

	res := &pb.PingNodeResponse{Ok: err == nil}

	if err != nil {
		return res, Error.Wrap(err)
	}
	return res, nil
}

// LookupNode triggers a Kademlia lookup and returns the node the network found.
func (srv *Inspector) LookupNode(ctx context.Context, req *pb.LookupNodeRequest) (*pb.LookupNodeResponse, error) {
	id, err := storj.NodeIDFromString(req.Id)
	if err != nil {
		return &pb.LookupNodeResponse{}, err
	}
	node, err := srv.dht.FindNode(ctx, id)
	if err != nil {
		return &pb.LookupNodeResponse{}, err
	}

	return &pb.LookupNodeResponse{
		Node: &node,
	}, nil
}

// DumpNodes returns all of the nodes in the routing table database.
func (srv *Inspector) DumpNodes(ctx context.Context, req *pb.DumpNodesRequest) (*pb.DumpNodesResponse, error) {
	nodes, err := srv.dht.DumpNodes(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.DumpNodesResponse{
		Nodes: nodes,
	}, nil
}

// NodeInfo sends a PING RPC to a node and returns its local info.
func (srv *Inspector) NodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	self := srv.dht.Local()

	info, err := srv.dht.FetchInfo(ctx, pb.Node{
		Id:      req.Id,
		Address: req.Address,
		Type:    self.Type,
	})
	if err != nil {
		return &pb.NodeInfoResponse{}, err
	}
	return &pb.NodeInfoResponse{
		Type:     info.GetType(),
		Operator: info.GetOperator(),
		Capacity: info.GetCapacity(),
		Version:  info.GetVersion(),
	}, nil
}

// GetBucketList returns the list of buckets with their routing nodes and their cached nodes
func (srv *Inspector) GetBucketList(ctx context.Context, req *pb.GetBucketListRequest) (*pb.GetBucketListResponse, error) {

	bucketIds, err := srv.dht.GetBucketIds()
	if err != nil {
		return nil, err
	}

	buckets := make([]*pb.GetBucketListResponse_Bucket, len(bucketIds))

	for i, b := range bucketIds {
		bucketID := keyToBucketID(b)
		routingNodes, _ := srv.dht.GetNodesWithinKBucket(bucketID)
		cachedNodes := srv.dht.GetCachedNodesWithinKBucket(bucketID)
		buckets[i] = &pb.GetBucketListResponse_Bucket{
			BucketId:     keyToBucketID(b),
			RoutingNodes: routingNodes,
			CachedNodes:  cachedNodes,
		}

	}
	return &pb.GetBucketListResponse{
		Buckets: buckets,
	}, nil
}
