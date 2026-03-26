'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import { Loader2, Key, Shield, RefreshCw } from 'lucide-react';

import { useAuth } from '@/lib/auth/context';
import { generateKeys, revokeAllTokens } from '@/lib/api/auth';
import { Header } from '@/components/layout/header';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';

export default function ProfilePage() {
  const { user, logout } = useAuth();
  const [isGeneratingKeys, setIsGeneratingKeys] = useState(false);
  const [isRevokingAll, setIsRevokingAll] = useState(false);
  const [generatedKeys, setGeneratedKeys] = useState<{
    words: string[];
    private_key: string;
  } | null>(null);

  const handleGenerateKeys = async () => {
    setIsGeneratingKeys(true);
    const result = await generateKeys();

    if (result.success) {
      setGeneratedKeys(result.data);
      toast.success('Encryption keys generated');
    } else {
      toast.error(result.error || 'Failed to generate keys');
    }
    setIsGeneratingKeys(false);
  };

  const handleRevokeAllSessions = async () => {
    setIsRevokingAll(true);
    const result = await revokeAllTokens();

    if (result.success) {
      toast.success('All sessions revoked');
      await logout();
    } else {
      toast.error(result.error || 'Failed to revoke sessions');
    }
    setIsRevokingAll(false);
  };

  return (
    <>
      <Header title="Profile" />
      <div className="flex flex-1 flex-col gap-6 p-6">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-bold tracking-tight">Profile</h1>
          <p className="text-muted-foreground">
            Manage your account settings and security
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          {/* Personal Information */}
          <Card>
            <CardHeader>
              <CardTitle>Personal Information</CardTitle>
              <CardDescription>Your personal details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>First Name</Label>
                  <Input
                    value={user?.data.firstName || ''}
                    disabled
                    className="bg-muted"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Last Name</Label>
                  <Input
                    value={user?.data.secondName || ''}
                    disabled
                    className="bg-muted"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Email</Label>
                <Input
                  value={user?.personal.mail || ''}
                  disabled
                  className="bg-muted"
                />
              </div>
              <div className="space-y-2">
                <Label>Phone</Label>
                <Input
                  value={user?.personal.phone || ''}
                  disabled
                  className="bg-muted"
                />
              </div>
            </CardContent>
          </Card>

          {/* Account Information */}
          <Card>
            <CardHeader>
              <CardTitle>Account Information</CardTitle>
              <CardDescription>Your account details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Username</Label>
                <Input
                  value={user?.app.username || ''}
                  disabled
                  className="bg-muted"
                />
              </div>
              <div className="space-y-2">
                <Label>User ID</Label>
                <Input
                  value={user?.app.userId?.toString() || ''}
                  disabled
                  className="bg-muted"
                />
              </div>
              <div className="space-y-2">
                <Label>Authentication Type</Label>
                <div className="flex items-center gap-2">
                  <Badge
                    variant={user?.app.authType === 'ldap' ? 'default' : 'secondary'}
                  >
                    <Shield className="size-3 mr-1" />
                    {user?.app.authType?.toUpperCase() || 'N/A'}
                  </Badge>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Encryption Keys */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Key className="size-5" />
                Encryption Keys
              </CardTitle>
              <CardDescription>
                Generate encryption keys for end-to-end encrypted messaging
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {generatedKeys ? (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label>Recovery Phrase (Keep this safe!)</Label>
                    <div className="p-3 bg-muted rounded-md text-sm font-mono break-all">
                      {generatedKeys.words.join(' ')}
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Private Key</Label>
                    <div className="p-3 bg-muted rounded-md text-sm font-mono break-all">
                      {generatedKeys.private_key}
                    </div>
                  </div>
                  <p className="text-sm text-destructive">
                    Warning: Save these keys securely. They will not be shown again.
                  </p>
                </div>
              ) : (
                <Button onClick={handleGenerateKeys} disabled={isGeneratingKeys}>
                  {isGeneratingKeys ? (
                    <>
                      <Loader2 className="size-4 animate-spin" />
                      Generating...
                    </>
                  ) : (
                    <>
                      <Key className="size-4" />
                      Generate Encryption Keys
                    </>
                  )}
                </Button>
              )}
            </CardContent>
          </Card>

          {/* Security */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="size-5" />
                Security
              </CardTitle>
              <CardDescription>
                Manage your account security settings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Separator />
              <div className="space-y-2">
                <h4 className="font-medium">Active Sessions</h4>
                <p className="text-sm text-muted-foreground">
                  Revoke all active sessions across all devices. You will need to
                  log in again.
                </p>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="destructive" disabled={isRevokingAll}>
                      {isRevokingAll ? (
                        <>
                          <Loader2 className="size-4 animate-spin" />
                          Revoking...
                        </>
                      ) : (
                        <>
                          <RefreshCw className="size-4" />
                          Revoke All Sessions
                        </>
                      )}
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Revoke All Sessions</AlertDialogTitle>
                      <AlertDialogDescription>
                        This will log you out of all devices and invalidate all
                        active tokens. You will need to log in again on all devices.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={handleRevokeAllSessions}>
                        Revoke All
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}
