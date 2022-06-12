package db

import (
	"context"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Added a flow struct
type FlowItem struct {
	/// From: "s" / "c" for server or client
	From string
	/// Data, in a somewhat reachable format
	Data string
	/// Data, as hex string
	Hex string
	/// Timestamp of the first packet in the flow (Epoch / ms)
	Time int
}

type FlowEntry struct {
	Src_port int
	Dst_port int
	Src_ip   string
	Dst_ip   string
	Time     int
	Duration int
	Inx      int
	Starred  bool
	Blocked  bool
	Filename string
	Suricata []int
	Flow     []FlowItem
	Tags     []string
}

type Database struct {
	client *mongo.Client
}

func ConnectMongo(uri string) Database {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}

	return Database{
		client: client,
	}
}

func (db Database) ConfigureDatabase() {
	db.InsertTag("flag-in")
	db.InsertTag("flag-out")
	db.InsertTag("fishy")
	db.InsertTag("steal")
	db.InsertTag("blocked")
	db.ConfigureIndexes()
}

func (db Database) ConfigureIndexes() {
	// create Index
	flowCollection := db.client.Database("pcap").Collection("pcap")

	_, err := flowCollection.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		// time index (range filtering)
		{
			Keys: bson.D{
				{"time", 1},
			},
		},
		// data index (context filtering)
		{
			Keys: bson.D{
				{"data", "text"},
			},
		},
		// port combo index (traffic correlation)
		{
			Keys: bson.D{
				{"src_port", 1},
				{"dst_port", 1},
			},
		},
	})

	if err != nil {
		panic(err)
	}
}

// Flows are either coming from a file, in which case we'll dedupe them by pcap file name.
// If they're coming from a live capture, we can do pretty much the same thing, but we'll
// just have to come up with a label. (e.g. capture-<epoch>)
// We can always swap this out with something better, but this is how flower currently handles deduping.
//
// A single flow is defined by a db.FlowEntry" struct, containing an array of flowitems and some metadata
func (db Database) InsertFlow(flow FlowEntry) {
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
func (db Database) InsertPcap(uri string) bool {
	files := db.client.Database("pcap").Collection("filesImported")
	shouldInsert := !db.ContainsPcap(uri)
	if shouldInsert {
		files.InsertOne(context.TODO(), bson.M{"file_name": uri})
	}
	return shouldInsert
}

func (db Database) ContainsPcap(uri string) bool {
	files := db.client.Database("pcap").Collection("filesImported")
	match := files.FindOne(context.TODO(), bson.M{"file_name": uri})
	return match.Err() != mongo.ErrNoDocuments
}

type FlowID struct {
	Src_port int
	Dst_port int
	Src_ip   string
	Dst_ip   string
	Time     time.Time
}

type Signature struct {
	MongoID primitive.ObjectID `bson:"_id,omitempty"`
	ID      int
	Msg     string
	Action  string
	Tag     string `bson:"omitempty"`
}

func (db Database) AddSignature(sig Signature) string {
	sigCollection := db.client.Database("pcap").Collection("signatures")

	// TODO; there's a bit of a race here, but I'm also racing to get this code working in time
	// for the next demo, so it all evens out.

	query := bson.M{
		"id":     sig.ID,
		"msg":    sig.Msg,
		"action": sig.Action,
	}

	var existing_sig Signature
	err := sigCollection.FindOne(context.TODO(), query).Decode(&existing_sig)
	if err != nil {
		// The signature does not appear in the DB yet. Let's add it.
		res, err := sigCollection.InsertOne(context.TODO(), query)
		if err != nil {
			log.Println("Rule add failed with error: ", err)
			return ""
		}
		ret := res.InsertedID.(primitive.ObjectID)
		return ret.Hex()
	} else {
		// The signature _does_ appear in the db. Let's return it's ID directly!
		return existing_sig.MongoID.Hex()
	}
}

func (db Database) AddSignatureToFlow(flow FlowID, sig Signature, window int) bool {
	// Add the signature to the collection
	sig_id := db.AddSignature(sig)
	if sig_id == "" {
		return false
	}

	// Find a flow that more or less matches the one we're looking for
	flowCollection := db.client.Database("pcap").Collection("pcap")
	epoch := int(flow.Time.UnixNano() / 1000000)
	query := bson.M{
		"src_port": flow.Src_port,
		"dst_port": flow.Dst_port,
		"src_ip":   flow.Src_ip,
		"dst_ip":   flow.Dst_ip,
		"time": bson.M{
			"$gt": epoch - window,
			"$lt": epoch + window,
		},
	}

	tags := []string{"fishy"}
	// TODO; pull this from metadata
	if strings.Contains(sig.Msg, "stolen flag") {
		tags = append(tags, "steal")
	}

	// A tag from the signature if it contained one
	if sig.Tag != "" {
		db.InsertTag(sig.Tag)
		tags = append(tags, sig.Tag)
	}

	var info bson.M
	// TODO; This can probably be done more elegantly, right?
	if sig.Action == "blocked" {
		info = bson.M{
			"$set": bson.M{
				"blocked": true,
			},
			"$addToSet": bson.M{
				"tags": bson.M{
					"$each": append(tags, "blocked"),
				},
				"suricata": sig_id,
			},
		}
	} else {
		info = bson.M{
			"$addToSet": bson.M{
				"tags": bson.M{
					"$each": tags,
				},
				"suricata": sig_id,
			},
		}
	}

	// enrich the flow with suricata information
	res, err := flowCollection.UpdateOne(context.TODO(), query, info)

	if err != nil {
		log.Println("Error occured while editing record:", err)
		return false
	}

	return res.MatchedCount > 0
}

func (db Database) InsertTag(tag string) {
	tagCollection := db.client.Database("pcap").Collection("tags")
	// Yeah this will err... A lot.... Two more dev days till Athens, this will do.
	tagCollection.InsertOne(context.TODO(), bson.M{"_id": tag})
}