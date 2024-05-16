## Term Cheatsheet

### Merkle Mountain Range (MMR)

The merkle mountain range is a type of hash tree.

It is our immutable audit log of datatrails events.

### MMR Index

variable: `mmrIndex`

The index of the node in the Merkle Mountain Range (MMR).

Starting from 0, incrementing for each new node added to the MMR.

Example:
```
              14
            /    \
           /      \
          /        \
         /          \
        6            13           21
      /   \        /    \
     2     5      9     12     17     20     24
    / \   / \    / \   /  \   /  \
   0   1 3   4  7   8 10  11 15  16 18  19 22  23   25
```

## MMR Position

variable: `mmrPosition`

The position of the node in the Merkle Mountain Range (MMR).

Starting from 1, incrementing for each new nod added to the MMR:

Example:
```
              15
            /    \
           /      \
          /        \
         /          \
        7            14           22
      /   \        /    \
     3     6      10     13    18     21     25
    / \   / \    / \   /  \   /  \
   1   2 4   5  8   9 11  12 16  17 19  20 23  24   26
```
It is a key property for all MMR implementations that the highest and left most position in the MMR when expresed in binary, is always all 1's. the number of 1's corresponds to the height. This is *why* we deal with both indices and positions. We use indices in all the context we can, because typically programers are dealing with an array. But ocasionaly, the binary 1's property is crucial.
### Leaf Index

variable: `leafIndex`

The index of the leaf node, in relation to other leaf nodes, in the Merkle Mountain Range (MMR).

Starting from 0, incrementing for each new leaf node added to the MMR.

Example:
```
              X
            /   \
           /     \
          /       \
         /         \
        X           X           X
      /   \       /   \
     X     X     X     X     X     X      X
    / \   / \   / \   / \   / \
   0   1 2   3 4   5 6   7 8   9 10  11 12  13   14
```

## MMR Height Index

The zero based height of the mmr. See also [mmr-math-cheatsheet](./mmr-math-cheatsheet.md)

variable: `heightIndex` or `g`

Aside: Why `g` ? so we can use `g+1`=`h` for the one based height

Example:
```
3             15
            /    \
           /      \
          /        \
         /          \
2       7            14           22
      /   \        /    \
1    3     6      10     13    18     21     25
    / \   / \    / \   /  \   /  \
0   1   2 4   5  8   9 11  12 16  17 19  20 23  24   26
```

At mmrPosition 15, `g` = 3

A leaf is defined where `g` = 0


## MMR Height

The one based height of the mmr. See also [mmr-math-cheatsheet](./mmr-math-cheatsheet.md)

variable: `height` or `h`

## Leaf Position

variable: `leafPosition`

The position of the leaf node, in relation to other leaf nodes, in the Merkle Mountain Range (MMR).

Starting from 1, incrementing for each new leaf node added to the MMR.

Example:
```
              X
            /   \
           /     \
          /       \
         /         \
        X           X           X
      /   \       /   \
     X     X     X     X     X     X      X
    / \   / \   / \   / \   / \
   1   2 3   4 5   6 7   8 9  10 11  12 13  14   15
```

### Leaf MMR Index

variable: `leafMMRIndex`

The index of a leaf node in the Merkle Mountain Range (MMR).

Starting from 0, incrementing for each new node added to the MMR,
ignoring intermediate nodes.

Example:
```
               X
            /     \
           /       \
          /         \
         /           \
        X             X           X
      /   \         /   \
     X     X       X     X      X     X      X
    / \   / \     / \   /  \   / \
   0   1 3   4   7   8 10  11 15 16 18  19 22  23   25
```

We are especialy strict about using `leafMMRIndex` for the index of the leaf in the full mmr. When talking about a leaf and its place in the mmr array, the inclusion of `MMRIndex` in the variable name is a 'must'

### Leaf Node

A leaf node is a node in the Merkle Mountain Range (MMR) that has a
height of 0. it is always on the bottom row of the MMR.

Example:
```
L = Leaf
N = Node

              N
            /   \
           /     \
          /       \
         /         \
        N           N           N
      /   \       /   \
     N     N     N     N     N     N     N
    / \   / \   / \   / \   / \
   L   L L   L L   L L   L L   L L   L L   L   L
```

### Intermediate Node

An intermediate node is a node in the Merkle Mountain Range (MMR) that
has a height of > 0. it is never on the bottom row of the MMR.

Example:
```
I = Intermediate
N = Node

              I
            /   \
           /     \
          /       \
         /         \
        I           I           I
      /   \       /   \
     I     I     I     I     I     I     I
    / \   / \   / \   / \   / \
   N   N N   N N   N N   N N   N N   N N   N   N
```

### Peak

A peak node is a node in the Merkle Mountain Range (MMR) that
has no node **directly** higher than itself.

Example:
```
P = Peak
N = Node

              P
            /   \
           /     \
          /       \
         /         \
        N           N           P
      /   \       /   \       /   \
     N     N     N     N     N     N     P
    / \   / \   / \   / \   / \   / \   / \
   N   N N   N N   N N   N N   N N   N N   N   P
```

### ID Timestamp

variable: `idTimestamp`

An ID Timestamp is a unique identifier based on a timestamp when the forestrie subsystem encounters an event.

We guarantee that the ID Timestamp increments for every committed event and is therefore monotonic.

ID Timestamp is equivalent to Snowflake ID.

### Snowflake ID

variable: `snowflakeID`

A snowflake ID is the mechanism which we use to ensure the time ordered uniqueness of the ID Timestamp.

Snowflake ID is equivalent to ID Timestamp.

### MMR Entry

variable: `mmrEntry`

An MMR Entry is the value given to a node in the Merkle Mountain Range.

For a leaf node this will be the hash of a datatrails events.
For an intermediate node this will be the hash of its position in the log and the two nodes below it

### MMR Data

variable: `mmrData`

MMR Data is all the mmr data, represented as a list of mmr entries
in the massif data.

Massif data format:
```
|--------|----------|---------|
| header | trieData | mmrData |
|--------|----------|---------|
```

### Trie Entry

variable: `trieEntry`

A Trie Entry is the companion data to a log entry.

Its current format is:

```
HASH(DOMAIN || LOGID || APPID) + IDTIMESTAMP
```

We hash the application key data in order to stop data leakage.

It is stored on the log in relation to the mmr entries as follows:

```
|--------|------------------------------|----------------------------|
| header | trieEntry0 ---> trieEntryMax | mmrEntry1 ---> mmrEntryMax |
|--------|------------------------------|----------------------------|
```

### Trie Index

variable: `trieIndex`

The index of a trie entry in the log.

Starting from 0, incrementing for each new trie entry added.

NOTE: the context to the trieIndex is the whole log, not relative to
the massif.

Trie index is equivalent to leaf index, as we only add a trie entry when
we add a leaf.

### Trie Key

variable: `trieKey`

A trie key is the key information for a leaf node.

The format of the trie key is:

```
HASH(DOMAIN || LOGID || APPID)
```

Where the `APPID` is the datatrails event identity.
Where the `LOGID` is the datatrails tenant identity.

We include the tenantID to ensure that the generated hash for the TrieKey is not the same
across the tenant boundary.

In order to recreate the hash, and find out if an event has been shared via the trieKey, would
require the eventID.

### Trie Data

variable: `trieData`

Trie Data is all the trie data, represented as a list of trie entries
in the massif data.

Massif data format:
```
|--------|----------|---------|
| header | trieData | mmrData |
|--------|----------|---------|
```

### Massif

A massif is a self contained portion of the Merkle Mountain Range (MMR).

The massif is stored as a single blob in blob storage. Therefore you can 
make the following statement: `massif is a synonm of blob in the context of forestrie`

The term massif means, amongs other things, "a sub range of mountains in a larger mountain range".

Its size depends on the maximum height, the MMR is allowed to get to,
before starting a new massif.

The massif is self contained because it contains all the leaf nodes and
intermediate nodes of its own portion of the MMR, but also the peaks of exactly and only those nodes it needs from *any* previous massif.

See the description of the peak stack maintenance in the mmrblob format document linked above
of previous massifs.

This property allows for proof generation of any node given ONLY
the information in the massif.

Example:
```
|           | 14           | 14            |
|           |              |               |
|           |              |               |
|           |              |               |
|.....6.....|......13......|.......21......| 2 maximum height
|   /   \   |    /    \    |      /   \    |
|  2     5  |   9     12   |    17     20  | 1 height
| / \   / \ |  / \    / \  |   /  \   /  \ |
|0   1 3   4| 7   8 10   11| 15   16 18  19| 0 height
| massif 0  |   massif 1   |    massif 2   |

Note: we ensure node 14 is also in massif 2. This allows
      massif 2 to be self contained and not rely on massif 1. 
```
