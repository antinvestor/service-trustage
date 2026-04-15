import 'package:flutter/material.dart';

/// Displays a JSON map as a formatted key-value list inside a card.
class JsonDataViewer extends StatelessWidget {
  const JsonDataViewer({
    super.key,
    required this.data,
    this.title,
  });

  final Map<String, dynamic> data;
  final String? title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    if (data.isEmpty) {
      return const SizedBox.shrink();
    }

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (title != null) ...[
              Text(title!, style: theme.textTheme.titleSmall),
              const SizedBox(height: 12),
            ],
            ...data.entries.map((entry) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: 140,
                        child: Text(
                          entry.key,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                      Expanded(
                        child: Text(
                          _formatValue(entry.value),
                          style: theme.textTheme.bodySmall,
                        ),
                      ),
                    ],
                  ),
                )),
          ],
        ),
      ),
    );
  }

  String _formatValue(dynamic value) {
    if (value == null) return '-';
    if (value is Map || value is List) return value.toString();
    return value.toString();
  }
}
