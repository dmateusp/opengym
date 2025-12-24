import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

function App() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      {/* Hero Section */}
      <div className="container mx-auto px-4 py-16">
        <div className="text-center mb-12">
          <h1 className="text-5xl font-bold text-gray-900 mb-4">
            opengym 🏐
          </h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Organize and participate in group sports with ease. Say goodbye to WhatsApp polls and hello to better game coordination.
          </p>
        </div>

        {/* Feature Cards */}
        <div className="grid md:grid-cols-3 gap-6 mb-12">
          <Card>
            <CardHeader>
              <CardTitle>Easy Organization</CardTitle>
              <CardDescription>Create games with all the details</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-gray-600">
                Set up games with date, time, location, and player limits. Track participation and waitlists effortlessly.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Smart Waitlists</CardTitle>
              <CardDescription>Never lose track of who's in</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-gray-600">
                Manage player participation with automatic waitlist tracking. See exactly where you stand.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Payment Tracking</CardTitle>
              <CardDescription>Know who paid, who didn't</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-gray-600">
                Split costs fairly among players and track payments. No more chasing people for money.
              </p>
            </CardContent>
          </Card>
        </div>

        {/* CTA Section */}
        <Card className="max-w-2xl mx-auto">
          <CardHeader>
            <CardTitle className="text-2xl text-center">Ready to organize your next game?</CardTitle>
            <CardDescription className="text-center">
              Join the community of players who've moved beyond WhatsApp polls
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex gap-4 justify-center">
              <Button size="lg" onClick={() => alert('Sign in coming soon!')}>
                Sign In
              </Button>
              <Button size="lg" variant="outline" onClick={() => alert('Learn more coming soon!')}>
                Learn More
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default App
