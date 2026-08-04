// SettingsSection — a titled card wrapper with an id anchor for the scroll-spy
// nav. Each settings domain renders one; the nav observes their ids.
import type { ReactNode } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'

interface Props {
  id: string
  title: string
  description?: string
  children: ReactNode
}

export function SettingsSection({ id, title, description, children }: Props) {
  return (
    <Card id={id} className="min-w-0 scroll-mt-20">
      <CardHeader className="min-w-0 gap-1.5">
        <CardTitle className="min-w-0 break-words">{title}</CardTitle>
        {description && (
          <CardDescription className="min-w-0 text-pretty leading-relaxed">
            {description}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="min-w-0 space-y-4">{children}</CardContent>
    </Card>
  )
}
