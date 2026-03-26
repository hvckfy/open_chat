'use client';

import { useRedirectIfAuthenticated } from '@/lib/auth/hooks';
import { LoginForm } from '@/components/auth/login-form';
import { Skeleton } from '@/components/ui/skeleton';

export default function LoginPage() {
  const { isLoading } = useRedirectIfAuthenticated('/');

  if (isLoading) {
    return (
      <div className="w-full max-w-md">
        <Skeleton className="h-[450px] w-full rounded-lg" />
      </div>
    );
  }

  return <LoginForm />;
}
