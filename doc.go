package curvecp

// Cookie format:
//
// 16 bytes: compressed nonce, prefix with "minute-k"
// 80 bytes: secretbox under minute-key, containing:
//     32 bytes: client short-term public key
//     32 bytes: server short-term secret key

// HELLO format:
//
// 0   : 8  : magic
// 8   : 16 : server extension
// 24  : 16 : client extension
// 40  : 32 : client short-term public key
// 72  : 64 : zero
// 136 : 8  : compressed nonce
// 144 : 80 : box C'->S containing:
//             0 : 64 : zero
//
// TOTAL: 224 bytes

// COOKIE format:
//
// 0  : 8   : magic
// 8  : 16  : client extension
// 24 : 16  : server extension
// 40 : 16  : compressed nonce
// 56 : 144 : box S->C' containing:
//             0  : 32 : server short-term public key
//             32 : 16 : compressed nonce
//             48 : 80 : minute-key secretbox containing:
//                        0  : 32 : client short-term public key
//                        32 : 32 : server short-term secret key
//
// TOTAL: 200 bytes

// INITIATE format:
//
// 0   : 8     : magic
// 8   : 16    : server extension
// 24  : 16    : client extension
// 40  : 32    : client short-term public key
// 72  : 96    : server's cookie
//                0  : 16 : compressed nonce
//                16 : 80 : minute-key secretbox containing:
//                           0  : 32 : client short-term public key
//                           32 : 32 : server short-term secret key
// 168 : 8     : compressed nonce
// 176 : 368+M : box C'->S' containing:
// 176 :          0   : 32  : client long-term public key
// 208 :          32  : 16  : compressed nonce
// 224 :          48  : 48  : box C->S containing:
//                             0 : 32 : client short-term public key
// 272 :          96  : 256 : server domain name
// 528 :          352 : M   : message
//
// TOTAL: 544+M bytes
//
// When Initiate passes validation in the packet pump, the C'->S' box
// gets replaced with the plaintext contents of the box. This is so
// that the conn handler doesn't need to double-decode retransmitted
// initiates. The absolute numbers for box elements are only valid
// once the packet has been verified and the plaintext content copied
// over the box.


// SERVER MESSAGE format:
//
// 0  : 8    : magic
// 8  : 16   : client extension
// 24 : 16   : server extension
// 40 : 8    : compressed nonce
// 48 : 16+M : box S'->C' containing:
//              0 : M : message
//
// TOTAL: 64+M bytes

// CLIENT MESSAGE format:
//
// 0   : 8    : magic
// 8   : 16   : server extension
// 24  : 16   : client extension
// 40  : 32   : client short-term public key
// 72  : 8    : compressed nonce
// 80  : 16+M : box C'->S' containing:
//               0 : M : message
//
// TOTAL: 96+M bytes
