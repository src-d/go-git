from six import string_types

from .go_object import GoObject


class StringMap(GoObject):
    def todict(self, cls):
        keys = self.keys()
        return {k: cls(self[k]) for k in keys}

    def __len__(self):
        return self.lib.c_std_map_len(self.handle)

    def keys(self):
        return self._string_slice(self.lib.c_std_map_keys_str(self.handle))

    def __getitem__(self, item):
        assert isinstance(item, string_types)
        go_key = self._string(item, self)
        return self.lib.c_std_map_get_str_obj(self.handle, go_key)

    def __delitem__(self, key):
        self.lib.c_std_map_set_str(
            self.handle, self._string(key, self), self.INVALID_HANDLE)

    def __setitem__(self, key, value):
        self.lib.c_std_map_set_str(
            self.handle, self._string(key, self), value.handle)
