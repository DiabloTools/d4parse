package d4

import (
	"github.com/Dakota628/d4parse/pkg/bin"
	"io"
	"os"
)

type TocEntry struct {
	SnoGroup SnoGroup
	SnoId    int32
	PName    int32

	SnoName string
}

func (t *TocEntry) UnmarshalBinary(r *bin.BinaryReader, o *Options) error {
	if err := r.Int32LE((*int32)(&t.SnoGroup)); err != nil {
		return err
	}

	if err := r.Int32LE(&t.SnoId); err != nil {
		return err
	}

	if err := r.Int32LE(&t.PName); err != nil {
		return err
	}

	return nil
}

type TocEntries map[SnoGroup]map[int32]string

func (e TocEntries) GetName(id int32, groups ...SnoGroup) (SnoGroup, string) {
	for _, group := range groups {
		if m, ok := e[group]; ok {
			if name, ok := m[id]; ok {
				return group, name
			}
		}
	}

	for group, m := range e {
		if name, ok := m[id]; ok {
			return group, name
		}
	}

	return SnoGroupUnknown, ""
}

type Toc struct {
	NumSnoGroups   int32
	EntryCounts    []int32 // n = numSnoGroups
	EntryOffsets   []int32 // n = numSnoGroups
	EntryUnkCounts []int32 // n = numSnoGroups
	Unk1           int32
	Entries        TocEntries
}

func (t *Toc) headerSize() int64 {
	return 4 + (int64(t.NumSnoGroups) * (4 + 4 + 4)) + 4
}

func (t *Toc) UnmarshalBinary(r *bin.BinaryReader, o *Options) error {
	if err := r.Int32LE(&t.NumSnoGroups); err != nil {
		return err
	}

	t.EntryCounts = make([]int32, t.NumSnoGroups)
	for i := int32(0); i < t.NumSnoGroups; i++ {
		if err := r.Int32LE(&t.EntryCounts[i]); err != nil {
			return err
		}
	}

	t.EntryOffsets = make([]int32, t.NumSnoGroups)
	for i := int32(0); i < t.NumSnoGroups; i++ {
		if err := r.Int32LE(&t.EntryOffsets[i]); err != nil {
			return err
		}
	}

	t.EntryUnkCounts = make([]int32, t.NumSnoGroups)
	for i := int32(0); i < t.NumSnoGroups; i++ {
		if err := r.Int32LE(&t.EntryUnkCounts[i]); err != nil {
			return err
		}
	}

	if err := r.Int32LE(&t.Unk1); err != nil {
		return err
	}

	// Move reader to after the header for relative seeks
	if err := r.Offset(t.headerSize()); err != nil {
		return err
	}

	var entry TocEntry
	t.Entries = make(map[SnoGroup]map[int32]string)
	for i := int32(0); i < t.NumSnoGroups; i++ {
		groupStartOffset := int64(t.EntryOffsets[i])
		groupEndOffset := groupStartOffset + (int64(t.EntryCounts[i]) * 12)
		if _, err := r.Seek(groupStartOffset, io.SeekStart); err != nil {
			return err
		}

		t.Entries[SnoGroup(i)] = make(map[int32]string)
		for j := int32(0); j < t.EntryCounts[i]; j++ {
			if err := entry.UnmarshalBinary(r, nil); err != nil {
				return err
			}

			if err := r.AtPos(groupEndOffset+int64(entry.PName), io.SeekStart, func(r *bin.BinaryReader) error {
				return r.NullTermString(&entry.SnoName)
			}); err != nil {
				return err
			}

			if _, ok := t.Entries[entry.SnoGroup]; !ok {
				t.Entries[entry.SnoGroup] = make(map[int32]string)
			}
			t.Entries[entry.SnoGroup][entry.SnoId] = entry.SnoName
		}
	}

	return nil
}

func ReadTocFile(path string) (Toc, error) {
	var toc Toc

	// Open file
	f, err := os.Open(path)
	if err != nil {
		return toc, err
	}
	defer f.Close()

	// Create binary reader
	r := bin.NewBinaryReader(f)

	// Unmarshal meta
	return toc, toc.UnmarshalBinary(r, nil)
}

// TODO: support payloads mapping
