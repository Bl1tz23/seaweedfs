package filer2

import (
	"context"
	"fmt"

	"github.com/chrislusf/seaweedfs/weed/glog"
	"github.com/chrislusf/seaweedfs/weed/pb/filer_pb"
)

func (f *Filer) DeleteEntryMetaAndData(ctx context.Context, p FullPath, isRecursive bool, ignoreRecursiveError, shouldDeleteChunks bool) (err error) {
	if p == "/" {
		return nil
	}

	entry, findErr := f.FindEntry(ctx, p)
	if findErr != nil {
		return findErr
	}

	var chunks []*filer_pb.FileChunk
	chunks = append(chunks, entry.Chunks...)
	if entry.IsDirectory() {
		// delete the folder children, not including the folder itself
		var dirChunks []*filer_pb.FileChunk
		dirChunks, err = f.doBatchDeleteFolderMetaAndData(ctx, entry, isRecursive, ignoreRecursiveError, shouldDeleteChunks)
		if err != nil {
			return fmt.Errorf("delete directory %s: %v", p, err)
		}
		chunks = append(chunks, dirChunks...)
		f.cacheDelDirectory(string(p))
	}
	// delete the file or folder
	err = f.doDeleteEntryMetaAndData(ctx, entry, shouldDeleteChunks)
	if err != nil {
		return fmt.Errorf("delete file %s: %v", p, err)
	}

	if shouldDeleteChunks {
		go f.DeleteChunks(chunks)
	}

	return nil
}

func (f *Filer) doBatchDeleteFolderMetaAndData(ctx context.Context, entry *Entry, isRecursive bool, ignoreRecursiveError, shouldDeleteChunks bool) (chunks []*filer_pb.FileChunk, err error) {

	lastFileName := ""
	includeLastFile := false
	for {
		entries, err := f.ListDirectoryEntries(ctx, entry.FullPath, lastFileName, includeLastFile, PaginationSize)
		if err != nil {
			glog.Errorf("list folder %s: %v", entry.FullPath, err)
			return nil, fmt.Errorf("list folder %s: %v", entry.FullPath, err)
		}
		if lastFileName == "" && !isRecursive && len(entries) > 0 {
			// only for first iteration in the loop
			return nil, fmt.Errorf("fail to delete non-empty folder: %s", entry.FullPath)
		}

		for _, sub := range entries {
			lastFileName = sub.Name()
			var dirChunks []*filer_pb.FileChunk
			if sub.IsDirectory() {
				dirChunks, err = f.doBatchDeleteFolderMetaAndData(ctx, sub, isRecursive, ignoreRecursiveError, shouldDeleteChunks)
			}
			if err != nil && !ignoreRecursiveError {
				return nil, err
			}
			if shouldDeleteChunks {
				chunks = append(chunks, dirChunks...)
			}
		}

		if len(entries) < PaginationSize {
			break
		}
	}

	f.cacheDelDirectory(string(entry.FullPath))

	glog.V(3).Infof("deleting directory %v", entry.FullPath)

	if storeDeletionErr := f.store.DeleteFolderChildren(ctx, entry.FullPath); storeDeletionErr != nil {
		return nil, fmt.Errorf("filer store delete: %v", storeDeletionErr)
	}
	f.NotifyUpdateEvent(entry, nil, shouldDeleteChunks)

	return chunks, nil
}

func (f *Filer) doDeleteEntryMetaAndData(ctx context.Context, entry *Entry, shouldDeleteChunks bool) (err error) {

	glog.V(3).Infof("deleting entry %v", entry.FullPath)

	if storeDeletionErr := f.store.DeleteEntry(ctx, entry.FullPath); storeDeletionErr != nil {
		return fmt.Errorf("filer store delete: %v", storeDeletionErr)
	}
	f.NotifyUpdateEvent(entry, nil, shouldDeleteChunks)

	return nil
}
