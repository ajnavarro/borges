package borges

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/src-d/go-git-fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/core-retrieval.v0/model"
	rrepository "gopkg.in/src-d/core-retrieval.v0/repository"
	"gopkg.in/src-d/core-retrieval.v0/test"
	"gopkg.in/src-d/go-billy.v3"
	"gopkg.in/src-d/go-billy.v3/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func TestArchiver(t *testing.T) {
	suite.Run(t, new(ArchiverSuite))
}

type ArchiverSuite struct {
	test.Suite
}

func (s *ArchiverSuite) SetupTest() {
	fixtures.Init()
	s.Suite.Setup()
}

func (s *ArchiverSuite) TearDownTest() {
	s.Suite.TearDown()
	fixtures.Clean()
}

func (s *ArchiverSuite) TestLastCommitDate() {
	for i, f := range fixtures.ByTag(".git") {
		s.T().Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			time, err := getLastCommitTime(newRepository(f))
			s.NoError(err)
			s.NotNil(time)
		})
	}
}

func (s *ArchiverSuite) TestReferenceUpdate() {
	for _, ct := range ChangesFixtures {
		if ct.FakeHashes {
			s.T().Run(ct.TestName, func(t *testing.T) {
				var obtainedRefs []*model.Reference
				for ic, cs := range ct.Changes { // emulate pushChangesToRootedRepositories() behaviour
					or := updateRepositoryReferences(ct.OldReferences, cs, ic)
					obtainedRefs = append(obtainedRefs, or...)
				}

				s.Equal(len(ct.NewReferences), len(obtainedRefs))
			})
		}
	}
}

func (s *ArchiverSuite) TestFixtures() {
	for _, ct := range ChangesFixtures {
		if ct.FakeHashes {
			continue
		}

		s.T().Run(ct.TestName, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			tmp, err := ioutil.TempDir(os.TempDir(),
				fmt.Sprintf("borge-tests%d", rand.Uint32()))
			require.NoError(err)
			defer func() { require.NoError(os.RemoveAll(tmp)) }()

			fs := osfs.New(tmp)

			rootedFs, err := fs.Chroot("rooted")
			require.NoError(err)
			txFs, err := fs.Chroot("tx")
			require.NoError(err)
			tmpFs, err := fs.Chroot("tmp")
			require.NoError(err)

			s := model.NewRepositoryStore(s.DB)
			tx := rrepository.NewSivaRootedTransactioner(rootedFs, txFs)
			a := NewArchiver(s, tx, tmpFs)

			a.Notifiers.Warn = func(j *Job, err error) {
				assert.NoError(err, "job: %v", j)
			}

			or, err := ct.OldRepository()
			require.NoError(err)

			err = withInProcRepository(or, func(url string) error {
				mr := model.NewRepository()
				mr.Endpoints = append(mr.Endpoints, url)
				mr.References = ct.OldReferences
				updated, err := s.Save(mr)
				require.NoError(err)
				require.False(updated)

				return a.Do(&Job{RepositoryID: uuid.UUID(mr.ID)})
			})
			require.NoError(err)

			checkNoFiles(t, txFs)
			checkNoFiles(t, tmpFs)

			nr, err := ct.NewRepository()
			require.NoError(err)

			checkReferences(t, nr, ct.NewReferences)
		})
	}
}

func newRepository(f *fixtures.Fixture) *git.Repository {
	fs := osfs.New(f.DotGit().Root())
	st, err := filesystem.NewStorage(fs)
	if err != nil {
		panic(err)
	}

	r, err := git.Open(st, fs)
	if err != nil {
		panic(err)
	}

	return r
}

func checkReferences(t *testing.T, obtained *git.Repository, refs []*model.Reference) {
	require := require.New(t)
	obtainedRefs := repoToMemRefs(t, obtained)
	expectedRefs := modelToMemRefs(t, refs)
	require.Equal(expectedRefs, obtainedRefs)
}

func modelToMemRefs(t *testing.T, refs []*model.Reference) memory.ReferenceStorage {
	require := require.New(t)
	m := memory.ReferenceStorage{}
	for _, ref := range refs {
		err := m.SetReference(ref.GitReference())
		require.NoError(err)
	}

	return m
}

func repoToMemRefs(t *testing.T, r *git.Repository) memory.ReferenceStorage {
	require := require.New(t)
	m := memory.ReferenceStorage{}
	iter, err := r.References()
	require.NoError(err)

	err = iter.ForEach(func(r *plumbing.Reference) error {
		if r.Type() != plumbing.HashReference {
			return nil
			//TODO: check this does not happen
		}

		return m.SetReference(r)
	})
	require.NoError(err)
	return m
}

func checkNoFiles(t *testing.T, fs billy.Filesystem) {
	require := require.New(t)

	fis, err := fs.ReadDir("")
	if !os.IsNotExist(err) {
		require.NoError(err)
	}

	for _, fi := range fis {
		require.True(fi.IsDir(), "unexpected file: %s", fi.Name())

		fsr, err := fs.Chroot(fi.Name())
		require.NoError(err)
		checkNoFiles(t, fsr)
	}
}
