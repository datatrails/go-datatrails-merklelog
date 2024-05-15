package snowflakeid

// This package implements the scheme for generating timestamp_committed as a
// time ordered snowflake id. As discussed on
// [forestrie-snowflakeid.md](https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/forestrie-snowflakeid.md)
//
// The following properties hold for the generated id's:
//
// * The id maps a time to the total ordering of all log entries created by DataTrails.
// * The order of the leaves over all merkle log trees matches the ordering of the ids
// * The order of the leaves in the tenant's merkle log matches the order of the snowflake ids for that tenant.
// * There is a 1:1 mapping from snowflake id to log position
// * A single snowflake id can not appear in > 1 tenant merkle logs
// * The 64 bit size of the id allows it to be used in contexts (like jwts and our timestamp_committed) which expect a time ordered integer timestamp-like value. And it allows it to be composed into a larger 256 bit key for data recovery and (limited) proof of exclusion purposes
//
// It also significantly helps disaster recovery procedures, see the linked doc
// for details
