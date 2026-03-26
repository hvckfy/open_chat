import { MessageSquare } from 'lucide-react';
import Link from 'next/link';

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-svh flex flex-col bg-background">
      {/* Header with logo */}
      <header className="flex items-center justify-center py-8">
        <Link href="/" className="flex items-center gap-2 text-foreground hover:text-foreground/80 transition-colors">
          <MessageSquare className="size-8" />
          <span className="text-2xl font-bold tracking-tight">OpenChat</span>
        </Link>
      </header>

      {/* Main content */}
      <main className="flex-1 flex items-start justify-center px-4 pb-12">
        {children}
      </main>

      {/* Footer */}
      <footer className="py-6 text-center text-sm text-muted-foreground">
        <p>Secure messaging for everyone</p>
      </footer>
    </div>
  );
}
