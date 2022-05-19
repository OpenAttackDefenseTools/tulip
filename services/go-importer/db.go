package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const WINDOW = 500 // ms

// Added a flow struct
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
	Src_port int
	Dst_port int
	Src_ip   string
	Dst_ip   string
	Time     int
	Duration int
	Inx      int
	Starred  int
	Filename string
	Flow     []flowItem
	Tag      string
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
		log.Println("Error occured while inserting record: ", err)
		log.Println("NO PCAP DATA WILL BE AVAILABLE FOR: ", flow.Filename)
	}
}

// Insert a new pcap uri, returns true if the pcap was not present yet,
// otherwise returns false
func (db database) InsertPcap(uri string) bool {
	files := db.client.Database("pcap").Collection("filesImported")

	match := files.FindOne(context.TODO(), bson.M{"file_name": uri})
	shouldInsert := match.Err() == mongo.ErrNoDocuments
	if shouldInsert {
		files.InsertOne(context.TODO(), bson.M{"file_name": uri})
	}
	return shouldInsert
}

type flowID struct {
	src_port int
	dst_port int
	src_ip   string
	dst_ip   string
	time     time.Time
}

func (db database) AddSignatureToFlow(suricata suricataLog) {

	// Find a flow that more or less matches the one we're looking for
	flow := suricata.flow
	flowCollection := db.client.Database("pcap").Collection("pcap")
	epoch := int(flow.time.UnixNano() / 1000000)
	query := bson.D{
		{"src_port", flow.src_port},
		{"dst_port", flow.dst_port},
		{"src_ip", flow.src_ip},
		{"dst_ip", flow.dst_ip},
		{"time", bson.D{
			{"$gt", epoch - WINDOW},
			{"$lt", epoch + WINDOW},
		}},
	}

	info := bson.M{"$set": bson.D{
		{"tag", "fishy"},
		{"suricata", suricata.signature},
	}}

	// enrich the flow with suricata information
	// TODO; update many -> update one
	res, err := flowCollection.UpdateMany(context.TODO(), query, info)

	if err != nil {
		log.Println("Error occured while editing record:", err)
		return
	}

	if res.MatchedCount > 0 {
		log.Println("Matched suricata signature", suricata.signature)
	}
}
