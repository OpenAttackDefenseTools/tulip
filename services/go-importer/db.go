package main

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// RDJ; Added a flow struct
type flowItem struct {
	/// From: "s" / "c" for server or client
	From string
	/// Data, in a somewhat reachable format
	Data string
	/// Data, as hex string
	Hex string
	/// Timestamp of the first packet in the flow (Epoch / ms)
	Time int
}

type flowEntry struct {
	Src_port      int
	Dst_port      int
	Src_ip        string
	Dst_ip        string
	Time          int
	Duration      int
	Inx           int
	Starred       int
	Contains_flag bool
	Filename      string
	Flow          []flowItem
}

type database struct {
	client *mongo.Client
}

func ConnectMongo(uri string) database {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}

	return database{
		client: client,
	}
}

// Flows are either coming from a file, in which case we'll dedupe them by pcap file name.
// If they're coming from a live capture, we can do pretty much the same thing, but we'll
// just have to come up with a label. (e.g. capture-<epoch>)
// We can always swap this out with something better, but this is how flower currently handles deduping.
//
// A single flow is defined by a "flowEntry" struct, containing an array of flowitems and some metadata
func (db database) InsertFlow(flow flowEntry) {
	flowCollection := db.client.Database("pcap").Collection("pcap")

	// TODO; use insertMany instead
	_, err := flowCollection.InsertOne(context.TODO(), flow)
	// check for errors in the insertion
	if err != nil {
		panic(err)
	}
}
