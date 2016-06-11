from six import string_types

from .go_object import GoObject
from .common import AuthMethod


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
        repo._strings[go_url] = c_url
        return repo

    @classmethod
    def NewPlain(cls):
        return Repository(cls.lib.c_NewPlainRepository())

    @property
    def remotes(self):
        map_handle = self.lib.c_Repository_get_Remotes(self.handle)
        return None

    @property
    def url(self):
        return self._string(self.lib.c_Repository_get_URL(self.handle))
