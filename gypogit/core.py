from six import python_2_unicode_compatible

from .go_object import GoObject


class Object(GoObject):
    pass


class ObjectStorage(GoObject):
    pass


class ObjectIter(GoObject):
    pass


@python_2_unicode_compatible
class ObjectType(int):
    INVALID = "<invalid>"
    MAP = (INVALID, "Commit", "Tree", "Blob", "Tag", INVALID, "OFSDelta",
           "REFDelta")

    def __str__(self):
        if self >= len(self.MAP):
            return self.MAP[0]
        return self.MAP[self]


for i, v in enumerate(ObjectType.MAP):
    if v != ObjectType.INVALID:
        setattr(ObjectType, v, ObjectType(i))
