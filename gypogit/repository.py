from six import string_types

from .go_object import GoObject
from .common import AuthMethod


class Repository(GoObject):
    @classmethod
    def Struct(cls):
        return Repository(cls.lib.c_Repository())

    @classmethod
    def New(cls, url, auth):
        assert isinstance(url, string_types)
        assert isinstance(auth, AuthMethod)
        go_url, c_url = cls.string(url)
        handle = cls.checked(cls.lib.c_NewRepository(go_url, auth.handle))
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
        return self.string(self.lib.c_Repository_get_URL(self.handle))
