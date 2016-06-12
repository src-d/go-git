from six import string_types

from .go_object import GoObject
from .common import AuthMethod
from .std import StringMap


class Remote(GoObject):
    pass


class Storage(GoObject):
    pass


class CommitIter(GoObject):
    pass


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
        return Storage(self.lib.c_Repository_get_Storage(self.handle))

    @Storage.setter
    def Storage(self, value):
        self.lib.c_Repository_set_Storage(self.handle, value.handle)

    def Commits(self):
        return CommitIter(self.lib.c_Repository_Commits(self.handle))
