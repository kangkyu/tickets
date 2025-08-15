# UMA E-Ticket Sales Platform Frontend

A modern React frontend for the UMA e-ticket sales platform, providing a clean, responsive interface for browsing events, purchasing tickets with UMA payments, and managing user tickets.

## 🚀 Features

### Core Functionality
- **Event Discovery**: Browse and search events with advanced filtering
- **Ticket Purchasing**: Complete UMA payment flow with Lightning Network
- **Real-time Payment Tracking**: Live payment status updates with polling
- **Ticket Management**: View, validate, and manage purchased tickets
- **QR Code Generation**: Lightning invoice QR codes for mobile payments
- **Responsive Design**: Mobile-first design optimized for all devices

### Technical Features
- **Modern React 18**: Built with latest React features and hooks
- **Type-safe Forms**: React Hook Form with Zod validation
- **Server State Management**: React Query for efficient data fetching
- **Real-time Updates**: WebSocket-like experience with polling
- **Error Handling**: Comprehensive error states and user feedback
- **Performance Optimized**: Code splitting, lazy loading, and memoization

## 🛠️ Tech Stack

- **Framework**: React 18 with Vite
- **Language**: JavaScript (ES6+)
- **Routing**: React Router DOM v6
- **State Management**: @tanstack/react-query
- **Forms**: React Hook Form + Zod validation
- **Styling**: Tailwind CSS
- **HTTP Client**: Axios with interceptors
- **Icons**: Lucide React
- **Date Handling**: date-fns
- **QR Codes**: qrcode library

## 📁 Project Structure

```
frontend/
├── src/
│   ├── components/          # React components
│   │   ├── EventList.jsx    # Event browsing and search
│   │   ├── EventCard.jsx    # Individual event display
│   │   ├── EventDetails.jsx # Detailed event view
│   │   ├── TicketPurchase.jsx # Purchase flow
│   │   ├── PaymentStatus.jsx   # Payment tracking
│   │   ├── TicketList.jsx      # User tickets
│   │   ├── QRCodeDisplay.jsx   # Lightning QR codes
│   │   └── Layout.jsx          # App layout
│   ├── hooks/              # Custom React hooks
│   │   ├── useEvents.js    # Event data management
│   │   ├── useTickets.js   # Ticket operations
│   │   └── usePayments.js  # Payment status
│   ├── services/           # API services
│   │   └── api.js         # HTTP client & endpoints
│   ├── utils/              # Utility functions
│   │   ├── formatters.js   # Data formatting
│   │   └── validators.js   # Form validation schemas
│   ├── config/             # Configuration
│   │   └── api.js         # API settings
│   ├── App.jsx             # Main app component
│   ├── main.jsx            # App entry point
│   └── index.css           # Global styles
├── package.json            # Dependencies
├── vite.config.js          # Vite configuration
├── tailwind.config.js      # Tailwind CSS config
└── index.html              # HTML template
```

## 🚀 Getting Started

### Prerequisites
- Node.js 16+ 
- npm or yarn

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd frontend
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Environment Configuration**
   Create a `.env` file in the root directory:
   ```env
   VITE_API_URL=http://localhost:8080
   ```

4. **Start development server**
   ```bash
   npm run dev
   ```

5. **Build for production**
   ```bash
   npm run build
   ```

## 🔧 Configuration

### API Configuration
The frontend connects to a backend API with the following endpoints:

#### Events API
- `GET /api/events` - List all events with filters
- `GET /api/events/:id` - Get event details
- `GET /api/events/search` - Search events

#### Tickets API
- `POST /api/tickets/purchase` - Purchase ticket
- `GET /api/tickets/:id/status` - Check ticket status
- `GET /api/tickets/user/:user_id` - Get user tickets
- `POST /api/tickets/validate` - Validate ticket

#### Payments API
- `GET /api/payments/:invoice_id/status` - Check payment status
- `POST /api/payments/create` - Create payment invoice
- `POST /api/payments/:invoice_id/cancel` - Cancel payment

### Environment Variables
- `VITE_API_URL`: Backend API base URL
- `VITE_MODE`: Application mode (development/production)

## 🎯 Key Components

### EventList Component
- Grid layout of event cards
- Advanced search and filtering
- Responsive design for mobile/desktop
- Loading states and error handling

### TicketPurchase Component
- Multi-step purchase form
- UMA address validation
- Real-time form validation
- Order summary sidebar

### PaymentStatus Component
- Lightning invoice QR code display
- Real-time payment polling
- Countdown timer for payment window
- Payment instructions and troubleshooting

### QRCodeDisplay Component
- Lightning invoice QR code generation
- Copy to clipboard functionality
- Mobile-optimized sizing
- Download QR code option

## 🔄 Data Flow

1. **Event Discovery**: User browses events with search/filter
2. **Event Selection**: User views event details and selects tickets
3. **Purchase Flow**: User fills form and submits purchase request
4. **Payment Generation**: Backend creates Lightning invoice
5. **Payment Display**: Frontend shows QR code and payment instructions
6. **Status Polling**: Real-time payment status updates
7. **Confirmation**: Ticket delivery upon successful payment

## 📱 Responsive Design

- **Mobile First**: Optimized for mobile devices
- **Touch Friendly**: Large touch targets and gestures
- **QR Code Sizing**: Optimized for phone camera scanning
- **Breakpoints**: Responsive grid layouts for all screen sizes

## 🎨 Design System

### Color Palette
- **Primary**: Blue (#3B82F6) for main actions
- **UMA**: Teal (#0EA5E9) for UMA-specific elements
- **Success**: Green (#10B981) for confirmations
- **Warning**: Orange (#F59E0B) for alerts
- **Error**: Red (#EF4444) for errors

### Typography
- **Font Family**: Inter (system fallback)
- **Headings**: Bold weights for hierarchy
- **Body**: Regular weight for readability

### Components
- **Cards**: Consistent shadow and border radius
- **Buttons**: Hover states and focus indicators
- **Forms**: Clear validation and error states
- **Loading**: Skeleton screens and spinners

## 🧪 Testing

The application is built with testing in mind:
- **Component Isolation**: Logic separated from presentation
- **Data Attributes**: `data-testid` for component identification
- **Mock APIs**: Consistent API mocking for tests
- **Form Validation**: Comprehensive validation testing

## 🚀 Performance Features

- **Code Splitting**: React.lazy() for route-based splitting
- **Image Optimization**: Lazy loading and fallbacks
- **Debounced Search**: Optimized search input handling
- **Memoization**: React.memo() for expensive components
- **Efficient Re-renders**: Optimized state updates

## 🔒 Security Considerations

- **Input Validation**: Client-side validation with Zod
- **XSS Prevention**: Safe HTML rendering
- **CSRF Protection**: Token-based API authentication
- **Secure Headers**: Content Security Policy ready

## 🌐 Browser Support

- **Modern Browsers**: Chrome 90+, Firefox 88+, Safari 14+
- **Mobile Browsers**: iOS Safari 14+, Chrome Mobile 90+
- **Progressive Enhancement**: Graceful degradation for older browsers

## 📋 Development Commands

```bash
# Development
npm run dev          # Start dev server
npm run build        # Build for production
npm run preview      # Preview production build
npm run lint         # Run ESLint

# Dependencies
npm install          # Install dependencies
npm update           # Update dependencies
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the API specifications

## 🔮 Future Enhancements

- **User Authentication**: Login/signup system
- **Push Notifications**: Payment status alerts
- **Offline Support**: Service worker for offline access
- **Analytics**: User behavior tracking
- **Internationalization**: Multi-language support
- **Dark Mode**: Theme switching
- **PWA**: Progressive web app features

---

Built with ❤️ for the UMA and Lightning Network communities.
