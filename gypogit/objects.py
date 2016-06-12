from dateutil import parser as iso8601parser
from six import python_2_unicode_compatible

from .go_object import GoObject
from .core import ObjectType


class Blob(GoObject):
    @classmethod
    def Decode(cls, data):
        go_data, c_data = cls._bytes(data)
        blob = Signature(cls.lib.c_Blob_Decode(go_data))
        blob._deps[go_data] = c_data
        return blob

    @property
    def Hash(self):
        return self._bytes(self.lib.c_Blob_get_Hash(self.handle), size=20)

    @property
    def ID(self):
        return self.Hash

    @property
    def Size(self):
        return self.lib.c_Blob_Size(self.handle)

    @property
    def Type(self):
        return ObjectType.Blob

    def Read(self):
        size, data = self._checked(self.lib.c_Blob_Read(self.handle), True)
        return self._bytes(data, size=size)


@python_2_unicode_compatible
class Signature(GoObject):
    @classmethod
    def Decode(cls, data):
        go_data, c_data = cls._bytes(data)
        sign = Signature(cls.lib.c_Signature_Decode(go_data))
        sign._deps[go_data] = c_data
        return sign

    @property
    def Name(self):
        return self._string(self.lib.c_Signature_Name(self.handle))

    @property
    def Email(self):
        return self._string(self.lib.c_Signature_Email(self.handle))

    @property
    def When(self):
        dts = self._string(self.lib.c_Signature_When(self.handle))
        return iso8601parser.parse(dts)

    def __str__(self):
        return "%s <%s> %s" % (self.Name, self.Email, self.When)