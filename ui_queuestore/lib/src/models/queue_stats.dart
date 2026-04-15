/// Live statistics for a queue.
class QueueStats {
  const QueueStats({
    required this.queueId,
    this.totalWaiting = 0,
    this.totalBeingServed = 0,
    this.averageWaitTime = 0,
    this.longestWaitTime = 0,
    this.completedToday = 0,
    this.cancelledToday = 0,
  });

  final String queueId;
  final int totalWaiting;
  final int totalBeingServed;
  final int averageWaitTime;
  final int longestWaitTime;
  final int completedToday;
  final int cancelledToday;

  factory QueueStats.fromJson(Map<String, dynamic> json) {
    return QueueStats(
      queueId: json['queue_id'] as String? ?? '',
      totalWaiting: json['total_waiting'] as int? ?? 0,
      totalBeingServed: json['total_being_served'] as int? ?? 0,
      averageWaitTime: json['average_wait_time'] as int? ?? 0,
      longestWaitTime: json['longest_wait_time'] as int? ?? 0,
      completedToday: json['completed_today'] as int? ?? 0,
      cancelledToday: json['cancelled_today'] as int? ?? 0,
    );
  }
}
