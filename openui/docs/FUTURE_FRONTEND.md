# Future Frontend Implementation Guide

This document outlines the frontend components and API integrations to implement when the messaging endpoints are added to `api.md`.

## Current Implementation Status

### Completed Features
- **Authentication**: LDAP and Local login/registration
- **Token Management**: Automatic refresh, session storage
- **Profile Management**: View user details, generate encryption keys
- **Security**: Revoke sessions, CSRF protection via cookies
- **UI Framework**: shadcn/ui with dark/light theme support
- **Navigation**: Responsive sidebar with user menu

### Placeholder Features (Ready for Extension)
- Dashboard stats cards (Messages, Contacts)
- Chats navigation item (disabled)
- Messaging section in dashboard

---

## Messaging Feature Implementation

When your backend adds messaging endpoints, implement these frontend features:

### 1. API Types to Add (`lib/api/types.ts`)

```typescript
// Message types
export interface Message {
  id: string;
  conversationId: string;
  senderId: number;
  content: string;
  timestamp: string;
  read: boolean;
  encrypted: boolean;
}

export interface Conversation {
  id: string;
  participants: User[];
  lastMessage: Message | null;
  unreadCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface SendMessageRequest {
  conversationId: string;
  content: string;
  encrypted?: boolean;
}

export interface CreateConversationRequest {
  participantIds: number[];
}
```

### 2. API Functions to Add (`lib/api/messages.ts`)

```typescript
import { api } from './client';
import type { Message, Conversation, SendMessageRequest, CreateConversationRequest } from './types';

// Get all conversations
export async function getConversations() {
  return api.get<Conversation[]>('/message/conversations');
}

// Get messages in a conversation
export async function getMessages(conversationId: string) {
  return api.get<Message[]>(`/message/conversation/${conversationId}`);
}

// Send a message
export async function sendMessage(data: SendMessageRequest) {
  return api.post<Message>('/message/send', data);
}

// Delete a message
export async function deleteMessage(messageId: string) {
  return api.delete<void>(`/message/${messageId}`);
}

// Create a new conversation
export async function createConversation(data: CreateConversationRequest) {
  return api.post<Conversation>('/message/conversation', data);
}

// Mark messages as read
export async function markAsRead(conversationId: string) {
  return api.post<void>(`/message/conversation/${conversationId}/read`);
}
```

### 3. WebSocket Integration (`lib/api/websocket.ts`)

```typescript
type MessageHandler = (message: Message) => void;
type StatusHandler = (status: 'connected' | 'disconnected' | 'error') => void;

class WebSocketClient {
  private ws: WebSocket | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  connect(url: string) {
    this.ws = new WebSocket(url);
    
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.notifyStatus('connected');
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data) as Message;
      this.messageHandlers.forEach(handler => handler(message));
    };

    this.ws.onclose = () => {
      this.notifyStatus('disconnected');
      this.attemptReconnect(url);
    };

    this.ws.onerror = () => {
      this.notifyStatus('error');
    };
  }

  private attemptReconnect(url: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      setTimeout(() => this.connect(url), 1000 * this.reconnectAttempts);
    }
  }

  private notifyStatus(status: 'connected' | 'disconnected' | 'error') {
    this.statusHandlers.forEach(handler => handler(status));
  }

  onMessage(handler: MessageHandler) {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onStatus(handler: StatusHandler) {
    this.statusHandlers.add(handler);
    return () => this.statusHandlers.delete(handler);
  }

  send(message: object) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  disconnect() {
    this.ws?.close();
    this.ws = null;
  }
}

export const wsClient = new WebSocketClient();
```

### 4. Components to Create

#### Chat List (`components/chat/chat-list.tsx`)
- List of conversations with avatars
- Unread message badges
- Last message preview
- Online status indicators
- Search/filter conversations

#### Chat Window (`components/chat/chat-window.tsx`)
- Message history with infinite scroll
- Message grouping by date
- Read receipts
- Typing indicators
- Attachment support (future)

#### Message Bubble (`components/chat/message-bubble.tsx`)
- Sent vs received styling
- Timestamp display
- Read status
- Message actions (delete, copy)

#### Message Input (`components/chat/message-input.tsx`)
- Text input with auto-resize
- Send button
- Attachment button (future)
- Emoji picker (future)

#### Contact List (`components/contacts/contact-list.tsx`)
- User search
- Add new contact
- Contact details

### 5. Pages to Create

#### Chats Page (`app/(protected)/chats/page.tsx`)
- Split view: conversation list + chat window
- Mobile: bottom navigation or drawer
- Real-time updates via WebSocket

#### Chat Detail Page (`app/(protected)/chats/[id]/page.tsx`)
- Full chat window for mobile
- Message history
- Input at bottom

### 6. Update Existing Components

#### Enable Chats Navigation (`components/layout/app-sidebar.tsx`)
Change:
```typescript
const futureNavItems = [
  {
    title: 'Chats',
    href: '/chats',  // Change from '#'
    icon: MessageSquare,
    disabled: false,  // Change from true
    // Remove badge
  },
];
```

#### Update Dashboard Stats (`app/(protected)/page.tsx`)
- Fetch actual message count
- Fetch contact count
- Display recent conversations

---

## Suggested Backend API Endpoints

For your reference, here are the suggested endpoints to implement:

```
# Conversations
GET    /api/message/conversations           # List all conversations
POST   /api/message/conversation            # Create new conversation
GET    /api/message/conversation/:id        # Get conversation details
DELETE /api/message/conversation/:id        # Delete conversation

# Messages
GET    /api/message/conversation/:id/messages  # Get messages (paginated)
POST   /api/message/send                       # Send message
DELETE /api/message/:id                        # Delete message
POST   /api/message/conversation/:id/read      # Mark as read

# Real-time
WS     /api/message/ws                      # WebSocket for real-time messages

# Contacts (optional)
GET    /api/account/contacts                # List contacts
POST   /api/account/contacts                # Add contact
DELETE /api/account/contacts/:id            # Remove contact
GET    /api/account/users/search?q=         # Search users
```

---

## Architecture Notes

### State Management
The current implementation uses React Context for auth state. For messaging, consider:
- **SWR** for conversation/message caching (already available)
- **React Context** for WebSocket connection state
- **Local state** for UI state (typing, selected conversation)

### Encryption
The `generateKeys` endpoint already provides encryption keys. When implementing E2E encryption:
1. Store private key securely (consider Web Crypto API)
2. Exchange public keys during conversation creation
3. Encrypt messages client-side before sending
4. Decrypt messages client-side after receiving

### Performance
- Implement message pagination (infinite scroll)
- Use virtual scrolling for long conversations
- Cache messages locally (IndexedDB)
- Optimize re-renders with React.memo

---

## File Structure After Messaging Implementation

```
app/
├── (protected)/
│   ├── chats/
│   │   ├── page.tsx              # Chats list page
│   │   └── [id]/
│   │       └── page.tsx          # Chat detail page
│   └── contacts/
│       └── page.tsx              # Contacts page

components/
├── chat/
│   ├── chat-list.tsx
│   ├── chat-window.tsx
│   ├── message-bubble.tsx
│   ├── message-input.tsx
│   └── typing-indicator.tsx
├── contacts/
│   ├── contact-list.tsx
│   ├── contact-card.tsx
│   └── add-contact-dialog.tsx

lib/
├── api/
│   ├── messages.ts               # Message API functions
│   └── websocket.ts              # WebSocket client
├── hooks/
│   ├── use-messages.ts           # Message hooks (SWR)
│   ├── use-conversations.ts      # Conversation hooks
│   └── use-websocket.ts          # WebSocket hook
```
