# Math, arithmetic and algorithms cheat sheet for MMR's


Some familiar context. See the [term cheatsheet](./term-cheatsheet.md)

```
4                        30


               14                        29
3            /  \                      /   \
            /    \                    /     \
           /      \                  /       \
          /        \                /         \
2       6 .      .  13             21          28
       /   \       /   \          /  \        /   \
1     2  |  5  |  9  |  12   |  17  | 20   | 24   | 27   |  --- massif tree line massif height index = 1
     / \ |/  \ | / \ |  /  \ | /  \ | / \  | / \  | / \  |
    0   1|3   4|7   8|10   11|15  16|18  19|22  23|25  26| MMR INDICES
    -----|-----|-----|-------|------|------|------|------|
    0   1|2   3|4   5| 6    7| 8   9|10  11|12  13|14  15| LEAF INDICES
    -----|-----|-----|-------|------|------|------|------|
      0  |  1  |  2  |  3    |   4  |   5  |   6  |   7  | MASSIF INDICES
    -----|-----|-----|-------|------|------|------|------|
```

## Notational conventions

* **n** shall be the `mmrPosition`, the `mmrSize` and the count of nodes.
* **g** shall be the *zero* based height index of an MMR. The height index of a leaf is 0.
* **h** shall be the *one* based height of an mmr
* **e** shall be the *zero* based `leafIndex`
* **f** shall be the *one* based `leafPosition` or count of leaves.

## Identities for power notation vs binary arithmetic

Shifting left is raising 2 to the power: $$2^n = 1 << n$$

Dividing by two is just shifting right. Or subtracting 1 from the existing left shift in a power expression.

$$\frac{2^n}{2} = 2^{n-1} = 1 << (n-1)$$

Multiplying by two is just shifting left. Or adding 1 to the existing left shift in a power expression.

$$2(2^n) = 2^{n+1} = 1 << (n+1) = 2 << n$$

Where the factors are powers of 2 these generalise as


$$\frac{2^n}{x} = 2^{n-\log_2(x)} = 1 << (n-\log_2(x))$$

$$x(2^n) = 2^{n+\log_2(x)} = 1 << (n+\log_2(x)) = (1<<\log_2(x)) << n$$

For example, where $$x = 8, n = 2$$

Then,

$$x(2^n) = 2^{n+\log_2(x)} = 1 << (n+\log_2(x)) = (1<<\log_2(x)) << n$$

$$= 8(2^2) = 2^{n+3} = 1 << (n+3) = (1<<3) << n$$

$$= 32 = 2^5 = 1 << 5 = (1 <<3) << 2$$

## Given a height or height index, how many nodes are there ?

$$n = 2^h-1 = (1 << h) - 1$$

$$n = 2^{g+1}-1 = (1 << (g+1)) - 1 = 2 << g - 1$$

[1]


## Given a node count, how many leaves are there ?

$$f=\frac{n+1}{2} = (n + 1) >> 1 = (e + 2) >> 1$$

[2]

## Given a height index, how many leaves are there ?

From [1] & [2] we can get

$$f= \frac{2^{g+1}-1+1}{2}$$

Then,

$$f = \frac{2^{g+1}}{2}$$

$$f = 2^{g+1-1}$$

$$f = 2^g = 1 << g$$

[3]


## Given a height, how many leaves are there ?

From [3] we get $$f = 2^g$$ so $$f = 2^{h-1} = 1 << (h - 1), h > 0$$