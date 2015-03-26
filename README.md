src
======

src 中的子包有些 fork 自 Go 标准库, 有些是新增的.
这些包多数出于特殊应用场景考虑, 或者需要增加一些通常不属于标准库提供的功能,
并需要触动标准库源码, 所以使用和标准库一致的目录结构.

请仔细了解各个子包的特性, 判断是否需要使用.

子包简述
========

go/parser
---------

增加 ScanLooker 支持, 最初是为了友好支持 minified Go.

go/printer/mini
---------

提供 minified Go 功能, 以紧凑格式输出 Go 源码.


LICENSE
=======

Copyright (c) 2015 The GoGraph Authors. All rights reserved. Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.