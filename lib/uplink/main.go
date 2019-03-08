// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package uplink

import (
	"context"
	"crypto"
	"crypto/x509"
	"errors"
	"io"
	"time"

	minio "github.com/minio/minio/cmd"
	"storj.io/storj/pkg/transport"

	"storj.io/storj/pkg/miniogw"
	"storj.io/storj/pkg/identity"
	"storj.io/storj/pkg/ranger"
	"storj.io/storj/pkg/storj"
)

// An Identity is a parsed leaf cert keypair with a certificate signed chain
// up to the self-signed CA with the node ID.
type Identity interface {
	NodeID() storj.NodeID
	Key() crypto.PrivateKey
	Certs() []x509.Certificate
}

// Caveat could be a read-only restriction, a time-bound
// restriction, a bucket-specific restriction, a path-prefix restriction, a
// full path restriction, etc.
type Caveat interface {
}

// A Macaroon represents an access credential to certain resources
type Macaroon interface {
	Serialize() ([]byte, error)
	Restrict(caveats ...Caveat) Macaroon
}

// Config holds the configs for the Uplink
type Config struct {
	// MaxBufferMem controls upload performance and is system-specific
	MaxBufferMem int

	// These should only be relevant for new files; these values for existing
	// files should come from the metainfo index. It's unlikely these will ever
	// change much.
	EncBlockSize  int
	MaxInlineSize int
	SegmentSize   int64
}

// Uplink represents the main entrypoint to Storj V3. An Uplink connects to
// a specific Satellite and caches connections and resources, allowing one to
// create sessions delineated by specific access controls.
type Uplink struct {
	ID      *identity.FullIdentity
	Session *Session
	SatelliteAddr string
}

// NewUplink creates a new Uplink
func NewUplink(ident *identity.FullIdentity, satelliteAddr string, cfg Config) *Uplink {
	return &Uplink{
		ID: id,
		SatelliteAddr: satelliteAddr,
	}
}

// BucketOpts holds the cipher, path, key, and enc. scheme for each bucket since they
// can be different for each
type BucketOpts struct {
	PathCipher       storj.Cipher
	EncPathPrefix    storj.Path
	Key              storj.Key
	EncryptionScheme storj.EncryptionScheme
}

// Access is all of the access information an application needs to store and
// retrieve data. Someone with a share may have no restrictions within a project
// (can create buckets, list buckets, list files, upload files, delete files,
// etc), may be restricted to a single bucket, may be restricted to a prefix
// within a bucket, or may even be restricted to a single file within a bucket.
// NB(dylan): You need an Access to start a Session
type Access struct {
	Permissions Macaroon

	// TODO: these should be per-bucket somehow maybe? oh man what a nightmare
	// Could be done via []Bucket with struct that has each of these
	// PathCipher       storj.Cipher // i.e. storj.AESGCM
	// EncPathPrefix    storj.Path
	// Key              storj.Key
	// EncryptionScheme storj.EncryptionScheme

	// Something like this?
	// TODO(dylan): Shouldn't actually use string, this is just a placeholder
	// until a more precise type is figured out - probably type Bucket
	Buckets map[string]BucketOpts
}

// ParseAccess parses a serialized Access
func ParseAccess(data []byte) (Access, error) {
	panic("TODO")
}

// Serialize serializes an Access message
func (a *Access) Serialize() ([]byte, error) {
	panic("TODO")
}

// Session represents a specific access session.
type Session struct {
	TransportClient *transport.Client
	Gateway         *minio.ObjectLayer
}

// NewSession creates a Session with an Access struct.
func (u *Uplink) NewSession(access Access) error {
	fi := &provider.FullIdentity{}

	tc := transport.NewClient(fi)

	// gateway := miniogw.NewGateway(ctx, fullIdentity)
	// layer := miniogw.NewGatewayLayer()

	u.Session = &Session{
		TransportClient: &tc,
		Gateway:         nil,
	}

	return nil
}

// GetBucket returns info about the requested bucket if authorized
func (s *Session) GetBucket(ctx context.Context, bucket string) (storj.Bucket,
	error) {

	// TODO: Wire up GetBucketInfo
	// info, err := s.Gateway.GetObject(ctx, bucket)
	// if err != nil {
	// 	return storj.Bucket{}, err
	// }

	return storj.Bucket{}, nil
}

// CreateBucketOptions holds the bucket opts
type CreateBucketOptions struct {
	PathCipher storj.Cipher
	// this differs from storj.CreateBucket's choice of just using storj.Bucket
	// by not having 2/3 unsettable fields.
}

// CreateBucket creates a new bucket if authorized
func (s *Session) CreateBucket(ctx context.Context, bucket string,
	opts *CreateBucketOptions) (storj.Bucket, error) {

	// s.Gateway.MakeBucketWithLocation(ctx, )

	return storj.Bucket{}, nil
}

// DeleteBucket deletes a bucket if authorized
func (s *Session) DeleteBucket(ctx context.Context, bucket string) error {
	return errors.New("Not implemented")
}

// ListBuckets will list authorized buckets
func (s *Session) ListBuckets(ctx context.Context, opts storj.BucketListOptions) (
	storj.BucketList, error) {
	return storj.BucketList{}, nil
}

// Access creates a new share, potentially further restricted from the Access used
// to create this session.
func (s *Session) Access(ctx context.Context, caveats ...Caveat) (Access, error) {
	panic("TODO")
}

// ObjectMeta represents metadata about a specific Object
type ObjectMeta struct {
	Bucket   string
	Path     storj.Path
	IsPrefix bool

	Metadata map[string]string

	Created  time.Time
	Modified time.Time
	Expires  time.Time

	Size     int64
	Checksum string

	// this differs from storj.Object by not having Version (yet), and not
	// having a Stream embedded. I'm also not sold on splitting ContentType out
	// from Metadata but shrugemoji.
}

// GetObject returns a handle to the data for an object and its metadata, if
// authorized.
func (s *Session) GetObject(ctx context.Context, bucket string, path storj.Path) (
	ranger.Ranger, ObjectMeta, error) {

	return nil, ObjectMeta{}, nil
}

// ObjectPutOpts controls options about uploading a new Object, if authorized.
type ObjectPutOpts struct {
	Metadata map[string]string
	Expires  time.Time

	// the satellite should probably tell the uplink what to use for these
	// per bucket. also these should probably be denormalized and defined here.
	RS            *storj.RedundancyScheme
	NodeSelection *miniogw.NodeSelectionConfig
}

// Upload uploads a new object, if authorized.
func (s *Session) Upload(ctx context.Context, bucket string, path storj.Path,
	data io.Reader, opts ObjectPutOpts) error {
	panic("TODO")
}

// DeleteObject removes an object, if authorized.
func (s *Session) DeleteObject(ctx context.Context, bucket string,
	path storj.Path) error {
	panic("TODO")
}

// ListObjectsField numbers the fields of list objects
type ListObjectsField int

const (
	ListObjectsMetaNone        ListObjectsField = 0
	ListObjectsMetaModified    ListObjectsField = 1 << iota
	ListObjectsMetaExpiration  ListObjectsField = 1 << iota
	ListObjectsMetaSize        ListObjectsField = 1 << iota
	ListObjectsMetaChecksum    ListObjectsField = 1 << iota
	ListObjectsMetaUserDefined ListObjectsField = 1 << iota
	ListObjectsMetaAll         ListObjectsField = 1 << iota
)

// ListObjectsConfig holds params for listing objects with the Gateway
type ListObjectsConfig struct {
	// this differs from storj.ListOptions by removing the Delimiter field
	// (ours is hardcoded as "/"), and adding the Fields field to optionally
	// support efficient listing that doesn't require looking outside of the
	// path index in pointerdb.

	Prefix    storj.Path
	Cursor    storj.Path
	Recursive bool
	Direction storj.ListDirection
	Limit     int
	Fields    ListObjectsFields
}

// ListObjectsFields is an interface that I haven't figured out yet
type ListObjectsFields interface{}

// ListObjects lists objects a user is authorized to see.
func (s *Session) ListObjects(ctx context.Context, bucket string,
	cfg ListObjectsConfig) (items []ObjectMeta, more bool, err error) {

	// TODO: wire up ListObjectsV2

	// s.Gateway.ListObjectsV2(bucket, cfg.Prefix, "/", cfg.Limit)
	panic("TODO")
}

// NewPartialUpload starts a new partial upload and returns that partial
// upload id
func (s *Session) NewPartialUpload(ctx context.Context, bucket string) (
	uploadID string, err error) {
	panic("TODO")
}

// TODO: lists upload ids
func (s *Session) ListPartialUploads() {
	panic("TODO")
}

// TODO: adds a new segment with given RS and node selection config
func (s *Session) PutPartialUpload() {
	panic("TODO")
}

// TODO: takes a path, metadata, etc, and puts all of the segment metadata
// into place. the object doesn't show up until this method is called.
func (s *Session) FinishPartialUpload() {
	panic("TODO")
}

// AbortPartialUpload cancels an existing partial upload.
func (s *Session) AbortPartialUpload(ctx context.Context,
	bucket, uploadID string) error {
	panic("TODO")
}
