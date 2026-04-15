/// Status of a queue item.
enum QueueItemStatus {
  waiting,
  serving,
  completed,
  cancelled,
  noShow,
  expired;

  static QueueItemStatus fromString(String value) {
    return switch (value) {
      'no_show' => QueueItemStatus.noShow,
      _ => QueueItemStatus.values.firstWhere(
          (s) => s.name == value,
          orElse: () => QueueItemStatus.waiting,
        ),
    };
  }

  String toApiString() {
    return switch (this) {
      QueueItemStatus.noShow => 'no_show',
      _ => name,
    };
  }
}

/// An item in a queue (a person/ticket waiting for service).
class QueueItem {
  const QueueItem({
    required this.id,
    required this.queueId,
    this.priority = 1,
    this.status = QueueItemStatus.waiting,
    this.ticketNo = '',
    this.category = '',
    this.customerId = '',
    this.counterId = '',
    this.servedBy = '',
    this.joinedAt,
    this.calledAt,
    this.serviceStart,
    this.serviceEnd,
    this.metadata,
    this.createdAt,
  });

  final String id;
  final String queueId;
  final int priority;
  final QueueItemStatus status;
  final String ticketNo;
  final String category;
  final String customerId;
  final String counterId;
  final String servedBy;
  final DateTime? joinedAt;
  final DateTime? calledAt;
  final DateTime? serviceStart;
  final DateTime? serviceEnd;
  final Map<String, dynamic>? metadata;
  final DateTime? createdAt;

  factory QueueItem.fromJson(Map<String, dynamic> json) {
    return QueueItem(
      id: json['id'] as String? ?? '',
      queueId: json['queue_id'] as String? ?? '',
      priority: json['priority'] as int? ?? 1,
      status: QueueItemStatus.fromString(json['status'] as String? ?? ''),
      ticketNo: json['ticket_no'] as String? ?? '',
      category: json['category'] as String? ?? '',
      customerId: json['customer_id'] as String? ?? '',
      counterId: json['counter_id'] as String? ?? '',
      servedBy: json['served_by'] as String? ?? '',
      joinedAt: json['joined_at'] != null
          ? DateTime.tryParse(json['joined_at'] as String)
          : null,
      calledAt: json['called_at'] != null
          ? DateTime.tryParse(json['called_at'] as String)
          : null,
      serviceStart: json['service_start'] != null
          ? DateTime.tryParse(json['service_start'] as String)
          : null,
      serviceEnd: json['service_end'] != null
          ? DateTime.tryParse(json['service_end'] as String)
          : null,
      metadata: json['metadata'] as Map<String, dynamic>?,
      createdAt: json['created_at'] != null
          ? DateTime.tryParse(json['created_at'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      if (priority != 1) 'priority': priority,
      if (category.isNotEmpty) 'category': category,
      if (customerId.isNotEmpty) 'customer_id': customerId,
      if (ticketNo.isNotEmpty) 'ticket_no': ticketNo,
      if (metadata != null) 'metadata': metadata,
    };
  }
}
