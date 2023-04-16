package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Movie struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title,omitempty"`
	Description string             `bson:"description,omitempty"`
	ReleaseDate string             `bson:"releasedate,omitempty"`
	Director    string             `bson:"director,omitempty"`
	Genre       string             `bson:"genre,omitempty"`
	Rating      float32            `bson:"rating,omitempty"`
	Runtime     int32              `bson:"runtime,omitempty"`
	Poster      string             `bson:"poster,omitempty"`
}
