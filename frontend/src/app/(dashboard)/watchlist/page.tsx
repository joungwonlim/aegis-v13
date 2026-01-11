'use client'

/**
 * 관심종목 페이지
 * SSOT: stocklist 모듈 사용
 *
 * 포트폴리오 연동 (보유종목 녹색점, 청산 모니터링 빨간점)은
 * StockDataTable 모듈 내부에서 자동 처리됨
 */

import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/components/ui/tabs'
import { Plus, Search, Loader2 } from 'lucide-react'
import {
  useStockList,
  useAddStock,
  useDeleteStock,
  StockListTable,
} from '@/modules/stocklist'

export default function WatchlistPage() {
  const [searchCode, setSearchCode] = useState('')

  // 관심종목 API 연동
  const { data: stocklistData, isLoading, error } = useStockList()
  const addStock = useAddStock()
  const deleteStock = useDeleteStock()

  // 데이터 분리
  const watchItems = stocklistData?.data?.watch ?? []
  const candidateItems = stocklistData?.data?.candidate ?? []
  const totalCount = watchItems.length + candidateItems.length

  // 종목 추가 핸들러
  const handleAdd = () => {
    if (!searchCode.trim()) return
    addStock.mutate({
      stock_code: searchCode.trim(),
      category: 'watch',
    })
    setSearchCode('')
  }

  // 종목 삭제 핸들러
  const handleDelete = (code: string) => {
    // 해당 코드의 종목 ID 찾기
    const item = [...watchItems, ...candidateItems].find(
      (i) => i.stock_code === code
    )
    if (item) {
      deleteStock.mutate(item.id)
    }
  }

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-destructive">
              관심종목을 불러오는데 실패했습니다.
            </p>
            <p className="text-sm text-muted-foreground mt-2">
              백엔드 서버 연결을 확인해주세요.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* 헤더 */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">관심종목</h1>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="종목코드 입력"
              value={searchCode}
              onChange={(e) => setSearchCode(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
              className="pl-9 w-40"
            />
          </div>
          <Button
            onClick={handleAdd}
            size="sm"
            disabled={addStock.isPending || !searchCode.trim()}
          >
            {addStock.isPending ? (
              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
            ) : (
              <Plus className="h-4 w-4 mr-1" />
            )}
            추가
          </Button>
        </div>
      </div>

      {/* 로딩 상태 */}
      {isLoading ? (
        <Card>
          <CardContent className="py-12 flex items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </CardContent>
        </Card>
      ) : (
        <Tabs defaultValue="watch" className="w-full">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="watch">
              관심종목 ({watchItems.length})
            </TabsTrigger>
            <TabsTrigger value="candidate">
              후보종목 ({candidateItems.length})
            </TabsTrigger>
          </TabsList>

          {/* 관심종목 탭 */}
          <TabsContent value="watch" className="mt-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-lg">
                  총 {watchItems.length}개 종목
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <StockListTable
                  items={watchItems}
                  onDelete={handleDelete}
                  emptyMessage="관심종목이 없습니다. 종목을 추가해보세요."
                />
              </CardContent>
            </Card>
          </TabsContent>

          {/* 후보종목 탭 */}
          <TabsContent value="candidate" className="mt-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-lg">
                  총 {candidateItems.length}개 종목
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <StockListTable
                  items={candidateItems}
                  onDelete={handleDelete}
                  emptyMessage="후보종목이 없습니다."
                />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}
    </div>
  )
}
