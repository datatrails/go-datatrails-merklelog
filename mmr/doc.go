package mmr

/*

# Motivation for the choice of MMR's

Merkle Binary Trees (not tries) are the simplest merkle structure. On its own
this is a great property. Merkle Mountain Ranges are a method of working with
binary merkles that has compelling benefits for the append-only-log ledger like
use case:

1. The structure is strictly append only and it is easy to prove this is the case
2. The position of a value in the tree is easily provable
3. There are efficient and simple answers for the problem of archiving historic
   state - it is not necessary to maintain the full log in hot storage (or at all)
   forever.
4. From one tree state to another, we can very simply demonstrate the
   consistency property - that everything that was in the previous tree is infact
   included in the new tree
5. This method combines very favourably with the cryptographic accumulator
   technique as
   [discussed](https://ethresear.ch/t/double-batched-merkle-log-accumulator/571)
   by Justin Drake in the context of ethereum. A typical problem with
   certificate transparancy style approaches (trilion) is that people have to
   continualy monitor the logs if they want to be sure that the log continues to
   contain their data. With MMR's combined with the approach in [1], there is
   still some 'root churn' but it is much slower and it is both sub linear
   (better than log(n) relative to the update frequency) and very well defined.
   People can reasonably 'check back later after quite some time'. And they do
   not need to 'stay live'.

All of this is achieved mostly due to one simple property: trees only grow to
the right and nothing is ever inserted. The Mountain Range comes from the fact
that this requires us to maintain multiple 'peaks'. With previous peaks being
combined as new elements are added. It turns out it is very directly possible to
manage those peaks based on knowing only the total number of elements in the
tree.

It is common to combine this sort of log with a trie for lookup and retrieval.
As the primary case for DataTrails is lookup single event for the purpose of receipt
generation, *and* as each event has a transaction id on it, we will have the
necessary context to efficiently find the event in the tree. We probably don't
need a trie in the classic patracia merkle tree sense.

# Approach, Sources & Background

The overall approach in this implementation follows the lead of the mimblewimble
rust implementation. It describes its overal structure [here](https://github.com/mimblewimble/grin/blob/0ff6763ee64e5a14e70ddd4642b99789a1648a32/core/src/core/pmmr.rs#L18)

In summary,

* The post order traversal (children first, left to right) of the MMR is
  identical to the natural append order of MMR nodes.
* Independent of the size of the tree (or its height), we can, from any position,
  'navigate' around the tree using simple binary arithmetic - the number of
  nodes to jump by is always some power of 2 relationship
* Because navigation is independent of the height and size of the tree we do not
  need to materialise the whole tree, or indeed any of it, in order to work with
  it.
* To take advantage of this we include a low level set of primitive functions
  which facilitate navigation around the binary tree realised as a flat sequence
  of positions
* We define a narrow interface for appending nodes and retrieving nodes based on
  their index (position - 1), that permits a variety of storage approaches.
* The low level api places a **burden of knowlege on the caller** in the interests
  of simplicity and efficiency. For example, calling a method to reach a sibling
  for a position that has no sibling will yeild nonsense results and the error
  will not be detected directly.
* Opinionated interfaces are provided on top of this, and those do provide
  appropriate safety rails.

## Post order traversal

Given a graph of 7 nodes like this,

       g
    c    f
  a   b d  e

The post order is children first, parents 'post', siblings left to right. so
flattening that tree in post order yeilds the labels above in series:

[a, b, c, d, e, f, g]
[1, 2, 3, 4, 5, 6, 7]

With the MMR's strictly append only nature, and its rule for back filling
Earlier peaks, this is the natural order of insertion of an MMR. To jump around
this sequence in post order, we can do some fairly straightforward binary
arithmetic, because it is a binary tree.

Note, for example, that 'jumping right' from c to its sibling f, is just

	3 + (2 << 1) - 1

And that no matter how large the tree grows, that operation from c moving right
to f remains the same.

This implementation draws from the following sources, making adjustmentst to
work with blob storage, and to deal with batch appends which are our common
case:

* https://github.com/mimblewimble/grin/blob/0ff6763ee64e5a14e70ddd4642b99789a1648a32/core/src/core/pmmr.rs#L606
* https://github.com/proofchains/python-proofmarshal/blob/master/proofmarshal/mmr.py
* https://github.com/jjyr/mmr.py/blob/master/mmr/mmr.py#L145
* https://github.com/zmitton/go-merklemountainrange/blob/master/mmr/mmr.go

Good general backgrounders are:
* https://neptune.cash/learn/mmr/
* https://docs.grin.mw/wiki/chain-state/merkle-mountain-range/
And peter todd's original case for using them in bitcoin:
* https://lists.linuxfoundation.org/pipermail/bitcoin-dev/2016-May/012715.html
And finally, Justin Drake's evolution of the approach which we may adopt:
* https://ethresear.ch/t/double-batched-merkle-log-accumulator/571

## IndexHeight

The extended remarks for the implementation lives in indexheight.go

For a python implementation see - https://github.com/proofchains/python-proofmarshal/blob/master/proofmarshal/mmr.py#L18
From https://github.com/mimblewimble/grin/blob/0ff6763ee64e5a14e70ddd4642b99789a1648a32/core/src/core/pmmr.rs#L606

"The height of a node in a full binary tree from its postorder traversal
index. This function is the base on which all others, as well as the MMR,
are built.

We first start by noticing that the insertion order of a node in a MMR [1]
is identical to the height of a node in a binary tree traversed in
postorder. Specifically, we want to be able to generate the following
sequence:

[0, 0, 1, 0, 0, 1, 2, 0, 0, 1, 0, 0, 1, 2, 3, 0, 0, 1, ...]

Which turns out to start as the heights in the (left, right, top)
-postorder- traversal of the following tree:

             3
           /   \
         /       \
       /           \
      2             2
    /  \          /  \
   /    \        /    \
  1      1      1      1
 / \    / \    / \    / \
0   0  0   0  0   0  0   0

If we extend this tree up to a height of 4, we can continue the sequence,
and for an infinitely high tree, we get the infinite sequence of heights
in the MMR.

So to generate the MMR height sequence, we want a function that, given an
index in that sequence, gets us the height in the tree. This allows us to
build the sequence not only to infinite, but also at any index, without the
need to materialize the beginning of the sequence.

To see how to get the height of a node at any position in the postorder
traversal sequence of heights, we start by rewriting the previous tree with
each the position of every node written in binary:

               1111
              /   \
            /       \
          /           \
        /               \
     111                1110
    /   \              /    \
   /     \            /      \
  11      110        1010     1101
 / \      / \       /  \      / \
1   10  100  101  1000 1001 1011 1100

The height of a node is the number of 1 digits on the leftmost branch of
the tree, minus 1. For example, 1111 has 4 ones, so its height is `4-1=3`.

To get the height of any node (say 1101), we need to travel left in the
tree, get the leftmost node and count the ones. To travel left, we just
need to subtract the position by it's most significant bit, mins one. For
example to get from 1101 to 110 we subtract it by (1000-1) (`13-(8-1)=5`).
Then to to get 110 to 11, we subtract it by (100-1) ('6-(4-1)=3`).

By applying this operation recursively, until we get a number that, in
binary, is all ones, and then counting the ones, we can get the height of
any node, from its postorder traversal position. Which is the order in which
nodes are added in a MMR.

[1]  https://github.com/opentimestamps/opentimestamps-server/blob/master/doc/merkle-mountain-range.md
"

## spur sum & navigating from massifs or leaf indices to storage blobs

This is additional to the various public references in order to accommodate
efficiently manageable backing storage in azure blobs/ s3 buckets etc

## SpurSum

	\       30

,                       /\          \
,                     /   \            \
,                    /     \ massif 2
,                  /        \              \
,                /           \          \    \
,  3        \   14   massif 1 \          \   29
,            \/    \           \           /    \
,   massif 0 /\     \           |         /  \     \
,           /   \    \          |        /     \     \
,  2      6 .    | .  13        |       21      |    28
,        /   \   |   /   \      |      / . \    |    /   \
,  1    2     5  |  9     12    |    17     20  |  24     27
,      / \  /  \ | / \    /  \  |   /  \   / \  |  / \ . ./ \
,     0   1 3   4| 7   8 10   11| 15   16 18  19| 22  23 25  26
.     0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15
                                   0 .  1 . 2  3 . 4 . 5 .6 . 7
,     | massif 0 |  massif 1 .  | massif 2      | massif 3

The basic deal with the algorithm is that the binary properties of the tree
allow a fairly efficient, log base 2 in the tree size, way to count and sum up
all the spurs.

 Each round i, starting at 1, calculates the *number* of spurs with height i,
 and multiplies by the length of that spur. the length of a spur is also its
 height which is also i.

 So for the above ascii, where the objective is to calculate the tree size or
 the massif start index knowing only the count of massifs we do

 h = mmr height - massif height = 4 - 1 = 3

 round 1: 1 << (3 - 1 -1) * i = 2 * 1  note the spurs ending at nodes 6 and 21 are the only length 1 spurs relative to the tree line
 round 2: 1 << (3 - 1 -2) * i = 1 * 2  note the spur ending at 14 is the only length 2 spur

for mmr height 3 we would get

3 - 1 = 2

r1: 1 << (2 - 1 -1) * 1 = 1

 Mathjax:

	\(sum = {\sum_{i=1}^{h-1}} 2^{h-1}/2^{i} * i\)

 => \(sum = {\sum_{i=1}^{h-1}} 2^{h-1-i} * i\)

## PrevSum, NextSum & TreeIndex

If we have the spur sum for height h, we can obtain the sum for the previous
height directly as:

    (sum - (height -1)) /2

Which is just:

    (sum - height + 1) >> 1


Intuitively, we are dropping height by first removing the nodes of the 'middle'
spur. THe remainder must be even and so dividing by two gives us the exact sum
for the next height down.

Similarly we can go the other way by doubling and adding height + 1

    (sum << 1) + height + 1

## MidLeafIndex

MidLeaf returns the 'mid' power of 2  leaf index which is closest to iLeaf.
This leaf is half way between the leaves on a power of 2 above and below i

$mid = 2^{\log_{2}(i)+1} - 2^{\log_{2}(i)-1}$

Which subtracting a bit wise power 2 round down from a bitwise power 2 round up

IMPORANT: i is a leaf index, relative to the leaves of the tree, *not* an MMR
position or index

    3        \   14           \             \ 29
              \/    \           \           / \      \
     massif 0 /\     \           |         /   \      \
             /   \    \          |        /     \      \
    2      6 .    | .  13        |       21      |     28
          /   \   |   /   \      |      / . \    |    /   \
    1    2     5  |  9     12    |    17     20  |  24     27
        / \  /  \ | / \    /  \  |   /  \   / \  |  / \ . ./ \
       0   1 3   4| 7   8 10   11| 15   16 18  19| 22  23 25  26
       0 . 1 2 . 3  4   5  6    7 . 8 .  9 10  11  12  13 14  15 LEAF INDICES

So for any leaf between 8 and 15 inclusive the answer is 12


*/
