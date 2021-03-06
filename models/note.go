package models

import (
	"bytes"
	"encoding/json"
	"github.com/boltdb/bolt"
	"strconv"
)

/**
 * DTO for a Note within a Notebook
 */
type Note struct {
	// TODO: explore allowing 'naming' notes within a notebook
	//Title   string `json:"title"`
	Id      uint64 `json:"id"`
	Content string `json:"content"`
}

/**
 * Returns whether or not note with a given id exists
 * in the given notebook or not
 */
func (db *DB) NoteExists(notebookName string, reqNoteId uint64) (bool, error) {
	noteExists := false
	err := db.View(func(tx *bolt.Tx) error {
		reqNoteIdBytes := []byte(strconv.FormatUint(reqNoteId, 10))
		notebookBucket := tx.Bucket([]byte("Notebook")).Bucket([]byte(notebookName))

		foundNoteIdBytes, _ := notebookBucket.Cursor().Seek(reqNoteIdBytes)
		if foundNoteIdBytes != nil && bytes.Equal(reqNoteIdBytes, foundNoteIdBytes) {
			noteExists = true
		}

		return nil
	})
	return noteExists, err
}

/**
 * Retrives note with a given id
 * param: uint64 noteId
 * return: (Note, error)
 */
func (db *DB) GetNote(notebookName string, reqNoteId uint64) (Note, error) {
	var note Note
	err := db.View(func(tx *bolt.Tx) error {
		reqNoteIdBytes := []byte(strconv.FormatUint(reqNoteId, 10))
		notebookBucket := tx.Bucket([]byte("Notebook")).Bucket([]byte(notebookName))

		foundNoteIdBytes, foundNoteContentBytes := notebookBucket.Cursor().Seek(reqNoteIdBytes)
		if foundNoteIdBytes != nil && bytes.Equal(reqNoteIdBytes, foundNoteIdBytes) {
			return json.Unmarshal(foundNoteContentBytes, &note)
		}

		return nil
	})
	return note, err
}

/**
 * Adds notes in the given notebook
 * notes' auto-increment 'Id' are generated and stored in the db by this method itself
 * param: string notebookName
 * param: ...Note notes
 * return: error
 */
func (db *DB) AddNotes(notebookName string, noteContents ...string) error {
	// create a bolt-db transaction with deferred-rollback
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// create or retrieve (2nd order) bucket with given notebookName
	notebookBucket, err := tx.Bucket([]byte("Notebook")).CreateBucketIfNotExists([]byte(notebookName))
	if err != nil {
		return err
	}

	// for each noteContent to be added
	for _, noteContent := range noteContents {
		// create Note object
		var note Note = Note{Content: noteContent}

		// gereate noteId
		noteId, err := notebookBucket.NextSequence()
		if err != nil {
			return err
		}
		note.Id = noteId

		// put JSON-marshalled noteContent into bolt-db bucket (of given Notebook) with noteId as key
		if encodedNote, err := json.Marshal(note); err != nil {
			return err
		} else if err := notebookBucket.Put([]byte(strconv.FormatUint(noteId, 10)), encodedNote); err != nil {
			return err
		}
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}

/**
 * Deletes notes with given ids from the given notebook
 * param: string notebookName
 * param: ...uint64 noteIds
 * return: error
 */
func (db *DB) DeleteNotes(notebookName string, noteIds ...uint64) error {
	// TODO: try to remove code-duplication: txn creation & notebook notebookBucket retrieval logic can be extracted out
	// create a bolt-db transaction with deferred-rollback
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// retrieve (2nd order) bucket with given notebookName
	notebookBucket := tx.Bucket([]byte("Notebook")).Bucket([]byte(notebookName))

	// for each noteId supplied
	for _, noteId := range noteIds {
		// delete the note with given noteId from notebook's bucket
		err = notebookBucket.Delete([]byte(strconv.FormatUint(noteId, 10)))
		if err != nil {
			return err
		}
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		return err
	}

	return err
}
