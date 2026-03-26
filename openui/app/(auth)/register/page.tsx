'use client';

import { useRedirectIfAuthenticated } from '@/lib/auth/hooks';
import { RegisterForm } from '@/components/auth/register-form';
import { Skeleton } from '@/components/ui/skeleton';

export default function RegisterPage() {
  const { isLoading } = useRedirectIfAuthenticated('/');

  if (isLoading) {
    return (
      <div className="w-full max-w-md">
        <Skeleton className="h-[650px] w-full rounded-lg" />
      </div>
    );
  }

  return <RegisterForm />;
}
