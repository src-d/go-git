from six import string_types

from .go_object import GoObject
from .common import AuthMethod
from .std import StringMap
from .core import Object, ObjectStorage
from .commit import Commit, CommitIter
from .tree import Tree
from .blob import Blob
from .tag import Tag, TagIter


class Repository(GoObject):
    @classmethod
    def Struct(cls):
        return Repository(cls.lib.c_Repository())

    @classmethod
    def New(cls, url, auth=None):
        assert isinstance(url, string_types)
        assert auth is None or isinstance(auth, AuthMethod)
        go_url, c_url = cls._string(url)
        handle = cls._checked(cls.lib.c_NewRepository(
            go_url, auth.handle if auth is not None else cls.INVALID_HANDLE))
        repo = Repository(handle)
        repo._deps[go_url] = c_url
        return repo

    @classmethod
    def NewPlain(cls):
        return Repository(cls.lib.c_NewPlainRepository())

    @property
    def Remotes(self):
        return StringMap(self.lib.c_Repository_get_Remotes(self.handle))

    @Remotes.setter
    def Remotes(self, value):
        self.lib.c_Repository_set_Remotes(self.handle, value.handle)

    @property
    def Url(self):
        return self._string(self.lib.c_Repository_get_URL(self.handle))

    @Url.setter
    def Url(self, value):
        self.lib.c_Repository_set_URL(self.handle, self._string(value, self))

    @property
    def Storage(self):
        return ObjectStorage(self.lib.c_Repository_get_Storage(self.handle))

    @Storage.setter
    def Storage(self, value):
        self.lib.c_Repository_set_Storage(self.handle, value.handle)

    def __getitem__(self, item):
        return self.Object(item)

    def Pull(self, remotename, branch):
        self._checked(self.lib.c_Repository_Pull(
            self.handle, self._string(remotename, self),
            self._string(branch, self)))

    def PullDefault(self):
        self._checked(self.lib.c_Repository_PullDefault(self.handle))

    def Commit(self, hash):
        return Commit(self._checked(self.lib.c_Repository_Commit(
            self.handle, self._hash(hash))))

    def Commits(self):
        return CommitIter(self.lib.c_Repository_Commits(self.handle))

    def Tree(self, hash):
        return Tree(self._checked(self.lib.c_Repository_Tree(
            self.handle, self._hash(hash))))

    def Blob(self, hash):
        return Blob(self._checked(self.lib.c_Repository_Blob(
            self.handle, self._hash(hash))))

    def Tag(self, hash):
        return Tag(self._checked(self.lib.c_Repository_Tag(
            self.handle, self._hash(hash))))

    def Tags(self):
        return TagIter(self._checked(self.lib.c_Repository_Tags(self.handle)))

    def Object(self, hash):
        return Object(self._checked(self.lib.c_Repository_Object(
            self.handle, self._hash(hash))))

    def _hash(self, h):
        if isinstance(h, string_types) and len(h) == 40:
            h = h.decode("hex")
        assert isinstance(h, (bytes, bytearray, memoryview))
        assert len(h) == 20
        return self._bytes(h, self)
