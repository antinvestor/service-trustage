/// Status of a service counter.
enum CounterStatus {
  open,
  closed,
  paused;

  static CounterStatus fromString(String value) {
    return CounterStatus.values.firstWhere(
      (s) => s.name == value,
      orElse: () => CounterStatus.closed,
    );
  }
}

/// A service counter/window attached to a queue.
class QueueCounter {
  const QueueCounter({
    required this.id,
    required this.queueId,
    required this.name,
    this.status = CounterStatus.closed,
    this.currentItemId = '',
    this.servedBy = '',
    this.totalServed = 0,
    this.categories = const [],
    this.createdAt,
    this.modifiedAt,
  });

  final String id;
  final String queueId;
  final String name;
  final CounterStatus status;
  final String currentItemId;
  final String servedBy;
  final int totalServed;
  final List<String> categories;
  final DateTime? createdAt;
  final DateTime? modifiedAt;

  factory QueueCounter.fromJson(Map<String, dynamic> json) {
    return QueueCounter(
      id: json['id'] as String? ?? '',
      queueId: json['queue_id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      status: CounterStatus.fromString(json['status'] as String? ?? ''),
      currentItemId: json['current_item_id'] as String? ?? '',
      servedBy: json['served_by'] as String? ?? '',
      totalServed: json['total_served'] as int? ?? 0,
      categories: (json['categories'] as List<dynamic>?)
              ?.map((e) => e.toString())
              .toList() ??
          const [],
      createdAt: json['created_at'] != null
          ? DateTime.tryParse(json['created_at'] as String)
          : null,
      modifiedAt: json['modified_at'] != null
          ? DateTime.tryParse(json['modified_at'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'name': name,
      if (categories.isNotEmpty) 'categories': categories,
    };
  }
}
