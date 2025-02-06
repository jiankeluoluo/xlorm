# Benchmark Report of Go ORM Frameworks

## Overview
This document compares the performance of three Go ORM frameworks (GORM, XLORM, and XORM) across various operational scenarios, including insertion, batch insertion, querying, batch querying, updating, batch updating, deletion, and transaction management. The testing environment is a Windows system with an AMD64 architecture, using an Intel(R) Core(TM) i5-7500 CPU @ 3.40GHz.

## Benchmark Results Comparison

### 1. Insert Operation
| Framework | Test Case               | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_Insert-4  | 1747 | 679999                    | 4798                     | 63                          |
| XLORM     | BenchmarkInsert-4       | 2164 | 575522                    | 1077                     | 28                          |
| XORM      | BenchmarkXORM_Insert-4  | 1951 | 581697                    | 2259                     | 43                          |

### 2. Batch Insert Operation
| Framework | Test Case                   | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-----------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_BatchInsert-4 | 1636 | 709971                    | 6595                     | 89                          |
| XLORM     | BenchmarkBatchInsert-4      | 1632 | 707283                    | 2713                     | 66                          |
| XORM      | BenchmarkXORM_BatchInsert-4 | 1864 | 620709                    | 3638                     | 85                          |

### 3. Query Operation
| Framework | Test Case               | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_Find-4    | 6344 | 192686                    | 4155                     | 64                          |
| XLORM     | BenchmarkFind-4         | 6679 | 175808                    | 2010                     | 39                          |
| XORM      | BenchmarkXORM_Find-4    | 5640 | 203428                    | 4274                     | 116                         |

### 4. Batch Query Operation
| Framework | Test Case                   | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-----------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_FindAll-4    | 584  | 2000131                   | 4864                     | 62                          |
| XLORM     | BenchmarkFindAll-4         | 601  | 1953513                   | 2456                     | 35                          |
| XORM      | BenchmarkXORM_FindAll-4    | 619  | 1986661                   | 3882                     | 86                          |

### 5. Update Operation
| Framework | Test Case               | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_Update-4  | 1506 | 833433                    | 6103                     | 69                          |
| XLORM     | BenchmarkUpdate-4       | 1634 | 739971                    | 1224                     | 26                          |
| XORM      | BenchmarkXORM_Update-4  | 6212 | 183031                    | 2577                     | 64                          |

### 6. Batch Update Operation
| Framework | Test Case                   | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-----------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_BatchUpdate-4 | 681  | 1746530                   | 13378                    | 167                         |
| XLORM     | BenchmarkBatchUpdate-4      | 1430 | 869917                    | 4173                     | 67                          |
| XORM      | BenchmarkXORM_BatchUpdate-4 | 1027 | 1244602                   | 6201                     | 160                         |

### 7. Delete Operation
| Framework | Test Case               | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|-------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_Delete-4  | 4236 | 278541                    | 5302                     | 62                          |
| XLORM     | BenchmarkDelete-4       | 7167 | 165919                    | 914                      | 18                          |
| XORM      | BenchmarkXORM_Delete-4  | 6818 | 178486                    | 2600                     | 69                          |

### 8. Transaction Operation
| Framework | Test Case                       | Runs | Time per Operation (ns/op) | Memory Allocation (B/op) | Allocation Count (allocs/op) |
|-----------|---------------------------------|------|---------------------------|--------------------------|-----------------------------|
| GORM      | BenchmarkGORM_Transaction-4     | 1572 | 751466                    | 5893                     | 65                          |
| XLORM     | BenchmarkTransaction-4          | 1419 | 818109                    | 2638                     | 75                          |
| XORM      | BenchmarkXORM_Transaction-4     | 1626 | 728734                    | 3149                     | 66                          |

## Summary
- **Performance**: XLORM shows the best performance in most operations, especially in querying and deletion. XORM performs best in update operations, while GORM is more stable in transaction management.
- **Memory Allocation**: XLORM has the lowest memory allocation and allocation counts in most scenarios.
- **Applicability**:
  - For projects involving frequent querying and deletion operations, XLORM is a good choice.
  - For scenarios requiring efficient update operations, XORM is more suitable.
  - GORM is stable in transaction handling and is suitable for applications requiring complex transaction management.

The above comparisons are based on the results of this benchmark test. When using these frameworks in actual projects, it is necessary to evaluate them comprehensively based on specific business requirements and other characteristics of the frameworks.