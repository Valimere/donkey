# Donkey
## Assignment:
Reddit, much like other social media platforms, provides a way for users to communicate their interests etc. For this exercise, we would like to see you build an application that listens to your choice of subreddits (best to choose one with a good amount of posts). You can use this link to help identify one that interests you.  We'd like to see this as a ~~.NET 6/7~~ (Confirmed can be Golang, Stephen)  application, and you are free to use any 3rd party libraries you would like.

Your app should consume the posts from your chosen subreddit in near real time and keep track of the following statistics between the time your application starts until it ends:

- Posts with most up votes
- Users with most posts

Your app should also provide some way to report these values to a user (periodically log to terminal, return from RESTful web service, etc.). If there are other interesting statistics you’d like to collect, that would be great. There is no need to store this data in a database; keeping everything in-memory is fine. That said, you should think about how you would persist data if that was a requirement.

To acquire near real time statistics from Reddit, you will need to continuously request data from Reddit's rest APIs.  Reddit implements rate limiting and provides details regarding rate limit used, rate limit remaining, and rate limit reset period via response headers.  Your application should use these values to control throughput in an even and consistent manner while utilizing a high percentage of the available request rate.

It’s very important that the various application processes do not block each other as Reddit can have a high volume on many of their subreddits.  The app should process posts as concurrently as possible to take advantage of available computing resources. While we are only asking to track a single subreddit, you should be thinking about his you could scale up your app to handle multiple subreddits.

While designing and developing this application, you should keep SOLID principles in mind. Although this is a code challenge, we are looking for patterns that could scale and are loosely coupled to external systems / dependencies. In that same theme, there should be some level of error handling and unit testing. The submission should contain code that you would consider production ready.

When you're finished, please put your project in a repository on either GitHub or Bitbucket and send us a link. Please be sure to provide guidance as to where the Reddit API Token values are located so that the team reviewing the code can replace/configure the value. After review, we may follow-up with an interview session with questions for you about your code and the choices made in design/implementation.

## 1. Understanding Requirements
Key Features:

-[ ] Real-time tracking of subreddit posts.
-[ ] Statistics: 
  - [ ] most upvoted posts
  - [ ] users with most posts.
-[ ] Reporting mechanism (e.g., logging or RESTful service).
-[ ] Handling Reddit API rate limits.
-[ ] Concurrency for high-volume data processing.
-[ ] Scalability for multiple subreddits.
-[ ] Adherence to SOLID principles.

### Optional Features:
-[ ] Additional interesting statistics (e.g., most discussed topics).
-[ ] Data persistence strategy for future extension.

## 2. Architecture Design
- Data Fetcher: Handles API requests to Reddit. Respects rate limits.
- Data Processor: Processes and analyzes incoming data.
- Statistics Tracker: Keeps in-memory records of required statistics.
- Reporter: Reports statistics via chosen methods.
- Concurrency Manager: Manages concurrent processing of posts.
- Scalability Consideration: Modular design to easily add more subreddits.
- Error Handling and Logging: Robust error handling and logging mechanisms.

## 3. Best Practices and SOLID Principles
- Single Responsibility: Each module has a single responsibility.
- Open/Closed: Easily extendable for more features like new statistics.
- Liskov Substitution & Interface Segregation: Use interfaces for modularity.
- Dependency Inversion: High-level modules should not depend on low-level modules.
## 4. Further Considerations
-  Scalability: Prepare for scaling up to handle multiple subreddits.
-  Data Persistence: Plan for a future database integration if needed.
-  Optimization: Monitor performance and optimize as needed.
## 5. Testing and Deployment
-  Unit Tests: Cover core functionality with tests.
-  Integration Tests: Ensure modules work together as expected.
