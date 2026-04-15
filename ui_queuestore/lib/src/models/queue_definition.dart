/// A queue definition with its configuration.
class QueueDefinition {
  const QueueDefinition({
    required this.id,
    required this.name,
    this.description = '',
    this.active = true,
    this.priorityLevels = 3,
    this.maxCapacity = 0,
    this.slaMinutes = 30,
    this.config,
    this.createdAt,
    this.modifiedAt,
  });

  final String id;
  final String name;
  final String description;
  final bool active;
  final int priorityLevels;
  final int maxCapacity;
  final int slaMinutes;
  final Map<String, dynamic>? config;
  final DateTime? createdAt;
  final DateTime? modifiedAt;

  factory QueueDefinition.fromJson(Map<String, dynamic> json) {
    return QueueDefinition(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      active: json['active'] as bool? ?? true,
      priorityLevels: json['priority_levels'] as int? ?? 3,
      maxCapacity: json['max_capacity'] as int? ?? 0,
      slaMinutes: json['sla_minutes'] as int? ?? 30,
      config: json['config'] as Map<String, dynamic>?,
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
      'description': description,
      'priority_levels': priorityLevels,
      'max_capacity': maxCapacity,
      'sla_minutes': slaMinutes,
      if (config != null) 'config': config,
    };
  }

  QueueDefinition copyWith({
    String? name,
    String? description,
    bool? active,
    int? priorityLevels,
    int? maxCapacity,
    int? slaMinutes,
    Map<String, dynamic>? config,
  }) {
    return QueueDefinition(
      id: id,
      name: name ?? this.name,
      description: description ?? this.description,
      active: active ?? this.active,
      priorityLevels: priorityLevels ?? this.priorityLevels,
      maxCapacity: maxCapacity ?? this.maxCapacity,
      slaMinutes: slaMinutes ?? this.slaMinutes,
      config: config ?? this.config,
      createdAt: createdAt,
      modifiedAt: modifiedAt,
    );
  }
}
